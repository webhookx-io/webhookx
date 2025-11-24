#!/bin/sh

echo "setting approle auth"
vault auth enable approle
vault policy write webhookx-read - <<EOF
path "secret/data/webhookx/*" {
  capabilities = ["read"]
}
EOF
vault write auth/approle/role/test-role \
    token_policies="webhookx-read" \
    secret_id_num_uses=0
vault write auth/approle/role/test-role/role-id role_id="test-role-id"
vault write auth/approle/role/test-role/custom-secret-id secret_id="test-secret-id"


echo "setting kubernetes auth"
vault auth enable kubernetes
vault write auth/kubernetes/config \
    token_reviewer_jwt=test \
    kubernetes_host=http://localhost:18888 \
    disable_local_ca_jwt=true
vault write auth/kubernetes/role/test-role \
    bound_service_account_names=webhookx \
    bound_service_account_namespaces=default \
    policies=webhookx-read \
    audience=webhookx
