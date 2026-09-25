#!/bin/bash

# Configuration
SOURCE_DIR="./sample-element" # Location of the sample element chart
DESTINATION_BASE="./stack" # Location of the elements

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# kebab-case to camelCase (cert-manager -> certManager). Keep in sync with chart.CamelFromKebab.
to_camel_case() {
    local input="$1"
    echo "$input" | awk -F'-' '{
        for(i=1; i<=NF; i++) {
            if(i==1) {
                printf "%s", tolower($i)
            } else {
                printf "%s", toupper(substr($i,1,1)) tolower(substr($i,2))
            }
        }
    }'
}

# Letters, digits and dashes only.
validate_name() {
    local name="$1"
    if [[ "$name" =~ ^[a-zA-Z0-9-]+$ ]]; then
        return 0
    else
        return 1
    fi
}

# Escape sed regex metacharacters.
escape_for_sed() {
    echo "$1" | sed 's/[[\.*^$()+?{|]/\\&/g'
}

echo -e "${GREEN}=== Element Chart Setup Script ===${NC}"
echo

if [ ! -d "$SOURCE_DIR" ]; then
    echo -e "${RED}Error: Source directory does not exist: $SOURCE_DIR${NC}"
    echo "Please update the SOURCE_DIR variable in this script to point to your Helm chart template."
    exit 1
fi

while true; do
    read -p "Enter the chart name (only letters, numbers, and dashes allowed): " CHART_NAME
    
    if [ -z "$CHART_NAME" ]; then
        echo -e "${RED}Error: Name cannot be empty${NC}"
        continue
    fi
    
    if validate_name "$CHART_NAME"; then
        break
    else
        echo -e "${RED}Error: Invalid name. Only letters, numbers, and dashes are allowed.${NC}"
    fi
done

read -p "Enter the chart description: " CHART_DESCRIPTION

# values.yaml keys the element under stack: by its camelCase name.
CAMEL_CASE_NAME=$(to_camel_case "$CHART_NAME")

DEST_DIR="$DESTINATION_BASE/$CHART_NAME"

if [ -d "$DEST_DIR" ]; then
    echo -e "${YELLOW}Warning: Directory $DEST_DIR already exists.${NC}"
    read -p "Do you want to overwrite it? (y/N): " OVERWRITE
    if [[ ! "$OVERWRITE" =~ ^[Yy]$ ]]; then
        echo "Operation cancelled."
        exit 0
    fi
    rm -rf "$DEST_DIR"
fi

echo
echo -e "${GREEN}Creating new Element chart...${NC}"
echo "- Name: $CHART_NAME"
echo "- Description: $CHART_DESCRIPTION"
echo "- CamelCase name: $CAMEL_CASE_NAME"
echo "- Destination: $DEST_DIR"
echo

echo "Copying template directory..."
cp -r "$SOURCE_DIR" "$DEST_DIR"
if [ $? -ne 0 ]; then
    echo -e "${RED}Error: Failed to copy template directory${NC}"
    exit 1
fi

CHART_YAML="$DEST_DIR/Chart.yaml"
if [ -f "$CHART_YAML" ]; then
    echo "Updating Chart.yaml..."
    
    TEMP_FILE=$(mktemp)
    
    # Escape special characters in the description for sed
    ESCAPED_DESC=$(escape_for_sed "$CHART_DESCRIPTION")
    
    # Replace the top-level name: and description: lines.
    awk -v name="$CHART_NAME" -v desc="$CHART_DESCRIPTION" '
        /^name:/ { print "name: " name; next }
        /^description:/ { print "description: " desc; next }
        { print }
    ' "$CHART_YAML" > "$TEMP_FILE"
    
    mv "$TEMP_FILE" "$CHART_YAML"
    
    if [ $? -ne 0 ]; then
        echo -e "${RED}Error: Failed to update Chart.yaml${NC}"
        exit 1
    fi
else
    echo -e "${YELLOW}Warning: Chart.yaml not found in $DEST_DIR${NC}"
fi

VALUES_YAML="$DEST_DIR/values.yaml"
if [ -f "$VALUES_YAML" ]; then
    echo "Updating values.yaml..."
    
    # The sample's stack key is the camelCase form of its own chart name.
    SOURCE_KEY=$(to_camel_case "$(awk '/^name:/ {print $2}' "$SOURCE_DIR/Chart.yaml")")
    sed -i.bak "s/\b$SOURCE_KEY\b/$CAMEL_CASE_NAME/g" "$VALUES_YAML"

    if [ $? -ne 0 ]; then
        echo -e "${RED}Error: Failed to update values.yaml${NC}"
        exit 1
    fi

    rm -f "$VALUES_YAML.bak"
else
    echo -e "${YELLOW}Warning: values.yaml not found in $DEST_DIR${NC}"
fi

echo
echo -e "${GREEN}✓ Element chart successfully created!${NC}"
echo "Location: $DEST_DIR"
echo
echo "Next steps:"
echo "  Update w/ configurations to deploy"
echo "  Update values.yaml"
