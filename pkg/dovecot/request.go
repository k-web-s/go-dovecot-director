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

type Request struct {
	AuthDomain    string `json:"auth_domain"`
	AuthUser      string `json:"auth_user"`
	AuthUsername  string `json:"auth_username"`
	Cert          string `json:"cert"`
	ClientId      string `json:"client_id"`
	Domain        string `json:"domain"`
	DomainFirst   string `json:"domain_first"`
	DomainLast    string `json:"domain_last"`
	Home          string `json:"home"`
	Lip           string `json:"lip"`
	LocalName     string `json:"local_name"`
	LoginDomain   string `json:"login_domain"`
	LoginUser     string `json:"login_user"`
	LoginUsername string `json:"login_username"`
	Lport         string `json:"lport"`
	MasterUser    string `json:"master_user"`
	Mech          string `json:"mech"`
	OrigDomain    string `json:"orig_domain"`
	OrigUser      string `json:"orig_user"`
	OrigUsername  string `json:"orig_username"`
	Password      string `json:"password"`
	Pid           string `json:"pid"`
	RealLip       string `json:"real_lip"`
	RealLport     string `json:"real_lport"`
	RealRip       string `json:"real_rip"`
	RealRport     string `json:"real_rport"`
	Rip           string `json:"rip"`
	Rport         string `json:"rport"`
	Secured       string `json:"secured"`
	Service       string `json:"service"`
	Session       string `json:"session"`
	SessionPid    string `json:"session_pid"`
	User          string `json:"user"`
	Username      string `json:"username"`
}
