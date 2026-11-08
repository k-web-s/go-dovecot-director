/*
Copyright (c) Richard Kojedzinszky <richard@kojedz.in>
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions
are met:
 1. Redistributions of source code must retain the above copyright
    notice, this list of conditions and the following disclaimer.
 2. Redistributions in binary form must reproduce the above copyright
    notice, this list of conditions and the following disclaimer in the
    documentation and/or other materials provided with the distribution.
 3. Neither the name of the University nor the names of its contributors
    may be used to endorse or promote products derived from this software
    without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE REGENTS AND CONTRIBUTORS “AS IS” AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
ARE DISCLAIMED.  IN NO EVENT SHALL THE REGENTS OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS
OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT
LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY
OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF
SUCH DAMAGE.
*/

package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"go-dovecot-director/pkg/allocator"
	"go-dovecot-director/pkg/pool"
)

const selectMapping = `SELECT backend,
       expires_at > NOW() AS valid,
       expires_at > NOW() + $2::interval AS fresh
  FROM mailbox_username_backend
 WHERE username = $1`

type postgresAllocator struct {
	pg      *pgxpool.Pool
	be      pool.Pool
	expiry  time.Duration
	refresh time.Duration
}

func New(pg *pgxpool.Pool, be pool.Pool, expiry, refresh time.Duration) allocator.Allocator {
	return &postgresAllocator{
		pg:      pg,
		be:      be,
		expiry:  expiry,
		refresh: refresh,
	}
}

// Allocate implements allocator.Allocator.
func (p *postgresAllocator) Allocate(ctx context.Context, username string) (backend string, err error) {
	var valid, fresh bool

	// Fast path: a single read, no transaction and no writes
	if err = p.pg.QueryRow(ctx, selectMapping, username, p.refresh).Scan(&backend, &valid, &fresh); err == nil {
		if valid && fresh {
			if available, _ := p.be.IsBackendAlive(ctx, backend); available {
				return
			}
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return
	}

	var tx pgx.Tx

	if tx, err = p.pg.Begin(ctx); err != nil {
		return
	}
	defer tx.Rollback(ctx)

	var rowExists bool

	// Lock the row and re-check its state
	if err = tx.QueryRow(ctx, selectMapping+" FOR UPDATE", username, p.refresh).Scan(&backend, &valid, &fresh); err == nil {
		rowExists = true
	} else if errors.Is(err, pgx.ErrNoRows) {
		err = nil
	} else {
		return
	}

	if rowExists {
		available, _ := p.be.IsBackendAlive(ctx, backend)

		switch {
		case valid && available && fresh:
			// Renewed concurrently, no write needed
		case valid && available:
			// Expiring soon, renew the lease only
			_, err = tx.Exec(ctx, "UPDATE mailbox_username_backend SET expires_at = NOW() + $2::interval WHERE username = $1", username, p.expiry)
		default:
			// Expired mapping or dead backend, reassign to a new backend
			if backend, err = p.be.GetBackend(ctx); err != nil {
				return
			}

			_, err = tx.Exec(ctx, "UPDATE mailbox_username_backend SET backend = $1, expires_at = NOW() + $3::interval WHERE username = $2", backend, username, p.expiry)
		}
	} else {
		// No mapping yet, create one
		if backend, err = p.be.GetBackend(ctx); err != nil {
			return
		}

		_, err = tx.Exec(ctx, "INSERT INTO mailbox_username_backend(backend, username, expires_at) VALUES ($1, $2, NOW() + $3::interval)", backend, username, p.expiry)

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23503" {
				err = nil

				return
			}
		}
	}

	if err != nil {
		return
	}

	err = tx.Commit(ctx)

	return
}
