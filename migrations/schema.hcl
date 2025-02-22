schema "public" {
}

table "operation_history" {
  schema = schema.public
  column "id" {
    null = false
    type = serial
  }
  column "operation" {
    null = false
    type = varchar(255)
  }
  column "project_id" {
    null = false
    type = varchar(255)
  }
  column "description" {
    null = true
    type = text
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("CURRENT_TIMESTAMP")
  }

  primary_key {
    columns = [column.id]
  }

  index "idx_operation_history_project_id" {
    columns = [column.project_id]
  }

  index "idx_operation_history_operation" {
    columns = [column.operation]
  }
}
