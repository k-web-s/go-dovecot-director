# go-dovecot-director

A [Dovecot Proxy](https://doc.dovecot.org/admin_manual/dovecot_proxy/) helper to map users to different backends. This application monitors backends running in Kubernetes. Simply, ready PODs are considered as live backends. They are monitored through Kubernetes endpoints.

Mapping is stored in PostgreSQL.

## Setting up

### Postgresql

Just simple table is needed for keeping records of username -> backend mappings. Howewer, is is __highly recommended__ to have
a foreign key on field `username` to valid users. For example, with postfixadmin the schema would be:

```sql
CREATE TABLE mailbox_username_backend (
    username character varying(255) NOT NULL PRIMARY KEY REFERENCES mailbox(username) ON DELETE CASCADE,
    backend character varying(255) NOT NULL,
    last_ts timestamp with time zone NOT NULL
);
```

### Kubernetes

`go-dovecot-director` will monitor a kubernetes service, technically an endpoint only. Thus, the needed RBAC rules are minimal:

```yaml
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: go-dovecot-director
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: go-dovecot-director
rules:
- apiGroups:
  - ""
  resources:
  - endpoints
  verbs:
  - get
  - list
  - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: go-dovecot-director
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: go-dovecot-director
subjects:
- kind: ServiceAccount
  name: go-dovecot-director
```

Then, the deployment needs to be told about the service/endpoint to monitor, and needs access to a postgresql table described in [postgresql](postgresql.md). This example monitors _dovecot-backend_ service in namespace _mail_.

```yaml
---
apiVersion: v1
kind: Secret
metadata:
  name: mail-database-secrets
stringDatadata:
  DATABASE_HOST: postgresql.database
  DATABASE_NAME: postfixadmin
  DATABASE_PASSWORD: password
  DATABASE_PORT: "5432"
  DATABASE_USER: postfixadmin
type: Opaque
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: go-dovecot-director
spec:
  selector:
    matchLabels:
      app: go-dovecot-director
  template:
    metadata:
      labels:
        app: go-dovecot-director
    spec:
      containers:
      - env:
        - name: NAMESPACE
          value: mail
        - name: SERVICE
          value: dovecot-backend
        envFrom:
        - secretRef:
            name: mail-database-secrets
        image: ghcr.io/k-web-s/go-dovecot-director
        name: go-dovecot-director
        resources:
          requests:
            cpu: 10m
            memory: 16Mi
        securityContext:
          capabilities:
            drop:
              - ALL
          allowPrivilegeEscalation: false
      securityContext:
        runAsNonRoot: true
      serviceAccountName: go-dovecot-director
---
apiVersion: v1
kind: Service
metadata:
  name: go-dovecot-director
spec:
  ports:
  - name: http
    port: 8080
    protocol: TCP
    targetPort: 8080
  selector:
    app: go-dovecot-director
```

### Dovecot

A frontend dovecot proxy should have the following configuration as its passdb and userdb driver:

```
passdb {
  driver = lua
  args = file=/etc/dovecot/proxy.lua blocking=yes
}

userdb {
  driver = lua
  args = file=/etc/dovecot/proxy.lua blocking=yes
}
```

The LUA script might look like:

```lua
--- simple module to proxy passdb and userdb requests to http
--- serializes request parameters as json, and returns the returned json
--- as a table

local json = require "cjson"

local url = "http://go-dovecot-director:8080"
local http_client = nil

function script_init()
  http_client = dovecot.http.client {
    timeout = 5000,
    max_attempts = 2,
  }

  return 0
end

local function proxy_lookup(uri, request)
    local http_request = http_client:request({ url = url .. uri, method = "POST" })
    http_request:set_payload(json.encode({user=request.user}))
    local http_response = http_request:submit()
    if http_response:status() ~= 200 then
        error("Invalid http status received")
    end
    local payload = http_response:payload()
    local resp = json.decode(payload)
    return resp.code, resp.attributes
end

function auth_passdb_lookup(request)
  return proxy_lookup("/auth_passdb_lookup", request)
end

function auth_userdb_lookup(request)
  return proxy_lookup("/auth_userdb_lookup", request)
end
```
