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

package dovecot

type PassdbResult int

// PassdbResult values from src/auth/passdb.h
const (
	PASSDB_RESULT_INTERNAL_FAILURE     PassdbResult = -1
	PASSDB_RESULT_SCHEME_NOT_AVAILABLE PassdbResult = -2
	PASSDB_RESULT_USER_UNKNOWN         PassdbResult = -3
	PASSDB_RESULT_USER_DISABLED        PassdbResult = -4
	PASSDB_RESULT_PASS_EXPIRED         PassdbResult = -5
	PASSDB_RESULT_NEXT                 PassdbResult = -6
	PASSDB_RESULT_PASSWORD_MISMATCH    PassdbResult = 0
	PASSDB_RESULT_OK                   PassdbResult = 1
)

type UserdbResult int

// UserdbResult values from src/auth/userdb.h
const (
	USERDB_RESULT_INTERNAL_FAILURE UserdbResult = -1
	USERDB_RESULT_USER_UNKNOWN     UserdbResult = -2
	USERDB_RESULT_OK               UserdbResult = 1
)
