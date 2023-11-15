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

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"go-dovecot-director/pkg/allocator"
	"go-dovecot-director/pkg/pool"
)

type postgresAllocator struct {
	pg *pgxpool.Pool
	be pool.Pool
}

func New(pg *pgxpool.Pool, be pool.Pool) allocator.Allocator {
	return &postgresAllocator{
		pg: pg,
		be: be,
	}
}

// Allocate implements allocator.Allocator.
func (p *postgresAllocator) Allocate(ctx context.Context, username string) (backend string, err error) {
	// One query without transaction, optimistic path
	if err = p.pg.QueryRow(ctx, "SELECT backend FROM mailbox_username_backend WHERE username = $1", username).Scan(&backend); err == nil {
		if available, _ := p.be.IsBackendAlive(ctx, backend); available {
			return
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return
	}

	return p.allocateTx(ctx, username)
}

// allocates in a transaction
func (p *postgresAllocator) allocateTx(ctx context.Context, username string) (backend string, err error) {
	var tx pgx.Tx

	if tx, err = p.pg.Begin(ctx); err != nil {
		return
	}
	defer tx.Rollback(ctx)

	var rowExists bool

	// Query and update timestamp
	if err = tx.QueryRow(ctx, "SELECT backend FROM mailbox_username_backend WHERE username = $1 FOR UPDATE", username).Scan(&backend); err == nil {
		if available, _ := p.be.IsBackendAlive(ctx, backend); available {
			err = tx.Commit(ctx)

			return
		}
		rowExists = true
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return
	}

	if backend, err = p.be.GetBackend(ctx); err != nil {
		return
	}

	if rowExists {
		_, err = tx.Exec(ctx, "UPDATE mailbox_username_backend SET backend = $1, last_ts = NOW() WHERE username = $2", backend, username)
	} else {
		_, err = tx.Exec(ctx, "INSERT INTO mailbox_username_backend(backend, username, last_ts) VALUES ($1, $2, NOW())", backend, username)
	}

	if err != nil {
		return
	}

	err = tx.Commit(ctx)

	return
}
