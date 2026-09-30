# Load the current aws login session for tools that do not read the CLI login cache.
# Callers must not echo the environment after sourcing this file.
load_aws_session() {
  eval "$(aws configure export-credentials --format env)"
  export AWS_EC2_METADATA_DISABLED=true
  export AWS_REGION=us-east-1
  export AWS_DEFAULT_REGION=us-east-1
}
