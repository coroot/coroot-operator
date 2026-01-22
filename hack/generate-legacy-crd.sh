#!/bin/bash
# Generate a legacy-compatible CRD by removing x-kubernetes-* extensions
# and adding nullable: true to top-level spec properties.
#
# This ensures compatibility with Kubernetes versions < 1.18 that don't
# support structural schema extensions.
#
# Usage: ./hack/generate-legacy-crd.sh <input-crd> <output-crd>
#
# Requires: yq v4+, jq

set -e

INPUT_FILE="$1"
OUTPUT_FILE="$2"

if [ -z "$INPUT_FILE" ] || [ -z "$OUTPUT_FILE" ]; then
    echo "Usage: $0 <input-crd> <output-crd>"
    exit 1
fi

if [ ! -f "$INPUT_FILE" ]; then
    echo "Error: Input file '$INPUT_FILE' not found"
    exit 1
fi

for cmd in yq jq; do
    if ! command -v $cmd &> /dev/null; then
        echo "Error: $cmd is required but not installed."
        exit 1
    fi
done

# Convert YAML to JSON, process with jq, convert back to YAML
yq -o=json "$INPUT_FILE" | jq '
# Recursive function to remove x-kubernetes-* fields
def remove_x_kubernetes:
  if type == "object" then
    # If has x-kubernetes-int-or-string, set type to string before removing
    (if has("x-kubernetes-int-or-string") and .["x-kubernetes-int-or-string"] == true
     then .type = "string"
     else . end) |
    # Remove all x-kubernetes-* keys
    with_entries(select(.key | startswith("x-kubernetes-") | not)) |
    # Recursively process all values
    with_entries(.value |= remove_x_kubernetes)
  elif type == "array" then
    map(remove_x_kubernetes)
  else
    .
  end;

# First remove x-kubernetes-* fields recursively
remove_x_kubernetes |
# Then add nullable: true to top-level spec properties only
.spec.versions[].schema.openAPIV3Schema.properties.spec.properties |=
  with_entries(.value += {"nullable": true})
' | yq -P > "$OUTPUT_FILE"

echo "Generated legacy CRD: $OUTPUT_FILE"
