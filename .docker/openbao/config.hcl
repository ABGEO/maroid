# The root token serves local development alone.
ui            = true
disable_mlock = true
api_addr      = "http://openbao:8200"

storage "file" {
  path = "/openbao/file/data"
}

listener "tcp" {
  address     = "0.0.0.0:8200"
  tls_disable = true
}

seal "static" {
  current_key_id = "maroid-local"
  current_key    = "env://BAO_STATIC_SEAL_KEY"
}

initialize "token" {
  request "root-token" {
    operation = "update"
    path      = "auth/token/create-orphan"
    data = {
      id                = "root-token"
      policies          = ["root"]
      no_default_policy = true
    }
  }
}
