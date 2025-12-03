# Checks if license files are present in all component directories.
REPO_DIR="$( cd "$(dirname "$( dirname "${BASH_SOURCE[0]}" )")" &> /dev/null && pwd )"

missing_licenses=false
MOD_TYPE_DIRS=("receiver")
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
    exit 1
fi

echo "✅ Licenses validated!"