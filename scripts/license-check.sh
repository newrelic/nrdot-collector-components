# Checks for the presence of LICENSE files in all components.
REPO_DIR="$( cd "$(dirname "$( dirname "${BASH_SOURCE[0]}" )")" &> /dev/null && pwd )"

missing_licenses=false
MOD_TYPE_DIRS=("receiver" "exporter" "connector" "extension" "processor")
for MOD_TYPE in "${MOD_TYPE_DIRS[@]}"; do
    component_dirs=$(find "$REPO_DIR/$MOD_TYPE" -mindepth 1 -maxdepth 1 -type d)
    for component in $component_dirs; do
        license=$(find "$component" -maxdepth 1 -type f -iname "LICENSE*")
        if [ -z "$license" ]; then
            component_path="${component#$REPO_DIR/}"
            echo "❌ License file missing in $component_path"
            missing_licenses=true
        fi
    done
done

if [ "$missing_licenses" = true ]; then
    echo "❌ License validation failed!"
    exit 1
fi

echo "✅ Licenses validated!"