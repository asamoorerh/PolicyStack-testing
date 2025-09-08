# PolicyStack Tools

This directory contains automation utilities for PolicyStack development and documentation. These tools streamline the creation of new policy elements and maintain synchronized documentation with your configuration.

## Table of Contents

- [Overview](#overview)
- [create-element.sh](#create-elementsh)
  - [Purpose](#purpose)
  - [Usage](#usage)
  - [Configuration](#configuration)
  - [Implementation Details](#implementation-details)
- [doc-generator.py](#doc-generatorpy)
  - [Purpose](#purpose-1)
  - [Architecture](#architecture)
  - [Usage](#usage-1)
  - [Comment Annotation System](#comment-annotation-system)
  - [Output Format](#output-format)
- [CI/CD Integration](#cicd-integration)
- [Best Practices](#best-practices)

## Overview

The PolicyStack tools provide:
- **Element Creation**: Automated scaffolding of new policy elements with proper structure
- **Documentation Generation**: Automatic extraction and formatting of inline documentation from YAML configurations
- **CI/CD Support**: GitHub Actions workflows for documentation validation and auto-generation

## create-element.sh

### Purpose

The element creation script automates the scaffolding of new PolicyStack elements by:
- Copying a template chart structure
- Replacing placeholder values with user-provided names
- Converting naming conventions (kebab-case to camelCase)
- Ensuring consistent element structure across the stack

### Usage

```bash
# Basic usage
./tools/create-element.sh

# The script will prompt for:
# 1. Chart name (alphanumeric and dashes only)
# 2. Chart description
```

#### Example

```bash
$ ./tools/create-element.sh
=== Element Chart Setup Script ===

Enter the chart name (only letters, numbers, and dashes allowed): security-baseline
Enter the chart description: Baseline security controls for cluster hardening

Creating new Element chart...
- Name: security-baseline
- Description: Baseline security controls for cluster hardening
- CamelCase name: securityBaseline
- Destination: ./stack/security-baseline

✓ Element chart successfully created!
Location: ./stack/security-baseline
```

### Configuration

The script uses configurable variables at the top of the file:

```bash
# Configuration
SOURCE_DIR="./sample-element"    # Template source directory
DESTINATION_BASE="./stack"       # Where elements are created
```

### Implementation Details

#### Name Transformation

The script performs automatic name transformation for values.yaml compatibility:

```bash
# Input: security-baseline
# Output in values.yaml: securityBaseline
```

This transformation ensures proper helm value referencing in the PolicyStack structure.

#### File Modifications

1. **Chart.yaml**: Updates name and description fields
2. **values.yaml**: Replaces all instances of `replaceMe` with the camelCase name
3. **Directory Structure**: Preserves the complete template structure including:
   - `templates/` directory with policy.yaml
   - `converters/` directory for manifest templates
   - `.gitignore` for helm dependencies

#### Error Handling

- Validates input names (alphanumeric + dashes only)
- Checks for existing directories with overwrite confirmation
- Verifies source template existence
- Provides colored output for better UX

## doc-generator.py

### Purpose

The documentation generator creates comprehensive markdown documentation from PolicyStack values.yaml files by:
- Parsing YAML with comment preservation at any nesting level
- Extracting `@description` and `@desc` annotations
- Generating structured documentation for all policy types
- Identifying configuration issues (orphaned policies)
- Creating an index of all documented elements

### Architecture

```
DocumentationGenerator
├── YAMLLoader                   # Comment-aware YAML parser
│   ├── _parse_with_comments()   # Extracts comments at all levels
│   ├── load()                   # Returns data + comment structure
│   └── get_description()        # Retrieves descriptions by path
│
├── Element Processing
│   ├── generate_element_docs()  # Main doc generation per element
│   ├── _generate_policy_section()
│   ├── _generate_config_policy_section()
│   ├── _generate_operator_policy_section()
│   └── _generate_certificate_policy_section()
│
└── Utilities
    ├── to_camel_case()          # Name conversion
    ├── _find_sub_policies()     # Relationship mapping
    ├── _find_orphaned_policies()# Validation
    └── _calculate_statistics()  # Summary generation
```

### Usage

```bash
# Generate documentation for all elements
python tools/doc-generator.py

# Custom directories
python tools/doc-generator.py --stack-dir ./custom-stack --output-dir ./custom-docs

# Single element documentation
python tools/doc-generator.py --element security-baseline
```

### Comment Annotation System

The generator supports hierarchical comment annotations using `@description:` or `@desc:` prefixes:

#### Basic Annotations

```yaml
# @description: Main policy for security compliance
security-policy:
  # @desc: Enable or disable this policy
  enabled: true
```

#### Nested Field Documentation

```yaml
configPolicies:
  - name: network-policy
    # @desc: Enforcement action when violations detected
    remediationAction: enforce
    
    # @description: Template configurations
    templateNames:
      # @desc: Ingress control template
      - name: ingress-rules
        complianceType: musthave
```

#### Array Item Documentation

```yaml
# @description: Approved versions for controlled upgrades
versions:
  # @desc: Initial stable release
  - v1.0.0
  # @desc: Security patch with CVE-2024-001 fix
  - v1.0.1
  # @desc: Performance improvements
  - v1.1.0
```

### Output Format

The generator produces:

#### Individual Element Documentation (`<element-name>.md`)

```markdown
# element-name - Policy Library Documentation

> Element description from Chart.yaml

*Generated: 2025-09-05 14:30:00*

## Component Configuration
| Parameter | Value | Description |
| --------- | ----- | ----------- |
| Component | `elementName` | Main component configuration |
| Enabled | `true` | Whether this component is enabled |

## Policies
### 📋 Policy: policy-name
> Policy description

[Detailed policy configuration tables...]

## PolicySets
[PolicySet configurations...]

## 📊 Summary
| Resource Type | Count |
| ------------- | ----- |
| Policies | 5 |
| Configuration Policies | 12 |
| **Total Resources** | **17** |
```

#### Index File (`docs/README.md`)

Lists all documented elements with links and provides comment notation guide.

## CI/CD Integration

The repository includes GitHub Actions workflows for documentation automation:

### docs-check.yml

Validates documentation is up-to-date on pull requests:

```yaml
- Runs on PR when stack/values files change
- Executes doc-generator.py
- Compares output (ignoring timestamp changes)
- Fails check if documentation needs updating
- Posts status comment with update instructions
```

### docs-update.yml

Auto-updates documentation when `update-docs` label is added:

```yaml
- Triggered by label addition
- Runs documentation generator
- Commits changes back to PR
- Removes label after completion
- Updates PR comment with status
```

## Best Practices

### For create-element.sh

1. **Naming Conventions**: Use kebab-case for element names (e.g., `security-baseline`, not `security_baseline`)
2. **Descriptions**: Write clear, concise descriptions that explain the element's purpose
3. **Template Maintenance**: Keep `sample-element` minimal but complete
4. **Version Control**: Commit new elements immediately after creation

### For doc-generator.py

1. **Comment Placement**: Place `@desc` comments immediately before the field they describe
2. **Description Quality**: Write descriptions that explain "why" not just "what"
3. **Consistency**: Use consistent terminology across descriptions
4. **Validation**: Run the generator locally before committing values changes
5. **Array Documentation**: Document array items when they have distinct purposes

### General

1. **Automation**: Use CI/CD workflows to maintain documentation consistency
2. **Review Process**: Include documentation review in PR reviews
3. **Versioning**: Document breaking changes in element configurations
4. **Testing**: Test element creation and documentation generation in development branches

## Contributing

When adding new features to these tools:

1. **Document the Feature**: Update this README with usage examples
2. **Add Tests**: Include test cases in your PR description
3. **Maintain Backwards Compatibility**: Don't break existing workflows
4. **Update CI/CD**: Adjust workflows if needed
5. **Consider Extensibility**: Design features to be easily extended
