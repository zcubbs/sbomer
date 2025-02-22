env "local" {
  src = "schema.hcl"
  dev = "postgres://postgres:postgres@localhost:5432/sbomer?sslmode=disable"
  url = "postgres://postgres:postgres@localhost:5432/sbomer?sslmode=disable"
  migration {
    dir = "file://migrations"
  }
  format {
    migrate {
      diff = "{{ sql . }}"
    }
  }
}
