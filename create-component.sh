#!/bin/bash

# Configuration
SOURCE_DIR="./sample-app" # Location of the sample app chart
DESTINATION_BASE="./stack" # Location of the components

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Function to convert string to camelCase
to_camel_case() {
    local input="$1"
    # Remove all dashes and convert to camelCase
    echo "$input" | awk -F'-' '{
        for(i=1; i<=NF; i++) {
            if(i==1) {
                # First word: lowercase
                printf "%s", tolower($i)
            } else {
                # Subsequent words: capitalize first letter, lowercase rest
                printf "%s", toupper(substr($i,1,1)) tolower(substr($i,2))
            }
        }
    }'
}

# Function to validate name (only alphanumeric and dashes)
validate_name() {
    local name="$1"
    if [[ "$name" =~ ^[a-zA-Z0-9-]+$ ]]; then
        return 0
    else
        return 1
    fi
}

# Function to escape special characters for sed
escape_for_sed() {
    echo "$1" | sed 's/[[\.*^$()+?{|]/\\&/g'
}

echo -e "${GREEN}=== Component Chart Setup Script ===${NC}"
echo

# Check if source directory exists
if [ ! -d "$SOURCE_DIR" ]; then
    echo -e "${RED}Error: Source directory does not exist: $SOURCE_DIR${NC}"
    echo "Please update the SOURCE_DIR variable in this script to point to your Helm chart template."
    exit 1
fi

# Get user inputs
# 1. Get name with validation
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

# 2. Get description
read -p "Enter the chart description: " CHART_DESCRIPTION

# Convert name to camelCase for values.yaml
CAMEL_CASE_NAME=$(to_camel_case "$CHART_NAME")

# Set destination directory
DEST_DIR="$DESTINATION_BASE/$CHART_NAME"

# Check if destination already exists
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
echo -e "${GREEN}Creating new Component chart...${NC}"
echo "- Name: $CHART_NAME"
echo "- Description: $CHART_DESCRIPTION"
echo "- CamelCase name: $CAMEL_CASE_NAME"
echo "- Destination: $DEST_DIR"
echo

# Copy the template directory
echo "Copying template directory..."
cp -r "$SOURCE_DIR" "$DEST_DIR"
if [ $? -ne 0 ]; then
    echo -e "${RED}Error: Failed to copy template directory${NC}"
    exit 1
fi

# Update Chart.yaml
CHART_YAML="$DEST_DIR/Chart.yaml"
if [ -f "$CHART_YAML" ]; then
    echo "Updating Chart.yaml..."
    
    # Create a temporary file
    TEMP_FILE=$(mktemp)
    
    # Escape special characters in the description for sed
    ESCAPED_DESC=$(escape_for_sed "$CHART_DESCRIPTION")
    
    # Update name and description in Chart.yaml
    # This handles various formats of Chart.yaml
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

# Update values.yaml
VALUES_YAML="$DEST_DIR/values.yaml"
if [ -f "$VALUES_YAML" ]; then
    echo "Updating values.yaml..."
    
    # Replace 'replaceMe' key with camelCase name
    # This handles both 'replaceMe:' as a key and 'replaceMe' as a value
    sed -i.bak "s/\breplaceMe\b/$CAMEL_CASE_NAME/g" "$VALUES_YAML"
    
    # Remove backup file
    rm -f "$VALUES_YAML.bak"
    
    if [ $? -ne 0 ]; then
        echo -e "${RED}Error: Failed to update values.yaml${NC}"
        exit 1
    fi
else
    echo -e "${YELLOW}Warning: values.yaml not found in $DEST_DIR${NC}"
fi

echo
echo -e "${GREEN}✓ Component chart successfully created!${NC}"
echo "Location: $DEST_DIR"
echo
echo "Next steps:"
echo "  Update w/ configurations to deploy"
echo "  Update values.yaml"
