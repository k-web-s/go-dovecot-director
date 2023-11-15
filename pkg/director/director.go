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

package director

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"go-dovecot-director/pkg/allocator"
	"go-dovecot-director/pkg/dovecot"
)

const (
	authPassdbLookupUri = "/auth_passdb_lookup"
	authUserdbLookupUri = "/auth_userdb_lookup"
)

type Director struct {
	allocator allocator.Allocator
}

func New(allocator allocator.Allocator) *Director {
	return &Director{
		allocator: allocator,
	}
}

func (d *Director) Serve(ctx context.Context, l net.Listener) error {
	mux := http.NewServeMux()

	mux.HandleFunc(authPassdbLookupUri, func(w http.ResponseWriter, r *http.Request) {
		d.authPassdbLookup(ctx, w, r)
	})
	mux.HandleFunc(authUserdbLookupUri, func(w http.ResponseWriter, r *http.Request) {
		d.authUserdbLookup(ctx, w, r)
	})

	server := http.Server{
		Handler: mux,
	}

	go func() {
		<-ctx.Done()

		server.Close()
	}()

	return server.Serve(l)
}

func (d *Director) redirect(ctx context.Context, w http.ResponseWriter, r *http.Request) (*dovecot.ResponseAttributes, bool) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		log.Print(err)

		w.WriteHeader(http.StatusInternalServerError)

		return nil, false
	}

	var authRequest dovecot.Request
	if err = json.Unmarshal(body, &authRequest); err != nil {
		log.Printf("Failed parsing request: %+v", err)

		w.WriteHeader(http.StatusBadRequest)

		return nil, false
	}

	i := 0
	for {
		if backend, err := d.allocator.Allocate(ctx, authRequest.User); err == nil {
			return &dovecot.ResponseAttributes{
				Nopassword: true,
				Proxy:      true,
				Host:       backend,
			}, true
		} else {
			log.Print(err)
		}

		i++

		if i == 5 {
			break
		}

		time.Sleep(time.Second)
	}

	w.WriteHeader(http.StatusInternalServerError)

	return nil, false
}

func sendResponse(w http.ResponseWriter, reply any) {
	replyb, err := json.Marshal(reply)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.Header().Add("Content-type", "application/json")
	_, err = w.Write(replyb)

	if err != nil {
		log.Print(err)
	}
}

func (d *Director) authPassdbLookup(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	attrs, success := d.redirect(ctx, w, r)
	if !success {
		return
	}

	response := &dovecot.PassdbResponse{
		Code:       dovecot.PASSDB_RESULT_OK,
		Attributes: attrs,
	}

	sendResponse(w, response)
}

func (d *Director) authUserdbLookup(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	attrs, success := d.redirect(ctx, w, r)
	if !success {
		return
	}

	response := &dovecot.UserdbResponse{
		Code:       dovecot.USERDB_RESULT_OK,
		Attributes: attrs,
	}

	sendResponse(w, response)
}
