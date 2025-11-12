#!/usr/bin/env python3
"""
Convert google-cloud-go .librarian/state.yaml to new librarian.yaml format.
"""

import yaml
import sys
from pathlib import Path

def convert_library(lib):
    """Convert a single library from old format to new format."""
    # Basic fields
    result = {
        'name': lib['id'],
        'version': lib.get('version'),
    }

    # Only add generate section if there are APIs
    if lib.get('apis'):
        generate = {
            'apis': []
        }

        # Convert APIs
        for api in lib['apis']:
            api_entry = {
                'path': api['path']
            }
            if api.get('service_config'):
                api_entry['service_config'] = api['service_config']
            generate['apis'].append(api_entry)

        # Convert preserve_regex to keep (if not empty)
        if lib.get('preserve_regex'):
            generate['keep'] = lib['preserve_regex']

        # Convert remove_regex to remove (if not empty)
        if lib.get('remove_regex'):
            generate['remove'] = lib['remove_regex']

        result['generate'] = generate

    # Note: tag_format is handled globally, not per-library in new format
    # Note: source_roots not needed in new format
    # Note: last_generated_commit not needed (sources section handles this)
    # Note: release_exclude_paths not in new format currently

    return result

def main():
    # Read the state.yaml
    state_path = Path.home() / 'code/googleapis/google-cloud-go/.librarian/state.yaml'
    with open(state_path) as f:
        state = yaml.safe_load(f)

    # Read the config.yaml for release_blocked info
    config_path = Path.home() / 'code/googleapis/google-cloud-go/.librarian/config.yaml'
    with open(config_path) as f:
        config = yaml.safe_load(f)

    # Extract release_blocked library IDs
    release_blocked = set()
    if config.get('libraries'):
        for lib in config['libraries']:
            if lib.get('release_blocked'):
                release_blocked.add(lib['id'])

    # Parse container image
    image_full = state.get('image', '')
    if '@sha256' in image_full:
        # Split on @sha256 to separate image from digest
        image_base = image_full.split('@')[0]
        # Get the last part as tag (though it's actually a digest)
        image = '/'.join(image_base.split('/')[:-1]) if '/' in image_base else image_base
        tag = image_full.split(':')[-1] if ':' in image_base else 'latest'
    else:
        # Fallback parsing
        parts = image_full.rsplit(':', 1)
        image = parts[0] if len(parts) > 1 else image_full
        tag = parts[1] if len(parts) > 1 else 'latest'

    # Build the new format
    output = {
        'version': 'v0.5.0',  # Current librarian version
        'language': 'go',
    }

    # Add container section
    if image:
        output['container'] = {
            'image': image if '@' not in image else image.split('@')[0],
            'tag': 'latest',  # Simplified for now
        }

    # Add generate section with defaults
    output['generate'] = {
        'output_dir': './',
        'defaults': {
            'transport': 'grpc+rest',
            'rest_numeric_enums': True,
            'release_level': 'stable',
        }
    }

    # Add release section
    output['release'] = {
        'tag_format': '{id}/v{version}'
    }

    # Convert libraries
    libraries = []
    for lib in state.get('libraries', []):
        converted = convert_library(lib)

        # Add comment for release_blocked libraries
        if lib['id'] in release_blocked:
            # Note: YAML comments need to be added manually or with special library
            converted['_comment'] = 'release_blocked: true (handwritten code)'

        libraries.append(converted)

    output['libraries'] = libraries

    # Write output
    output_path = Path('data/go/librarian.yaml')
    output_path.parent.mkdir(parents=True, exist_ok=True)

    with open(output_path, 'w') as f:
        # Write with explicit formatting
        yaml.dump(output, f,
                 default_flow_style=False,
                 sort_keys=False,
                 allow_unicode=True,
                 width=120)

    print(f"Converted {len(libraries)} libraries to {output_path}")
    print(f"Release-blocked libraries: {len(release_blocked)}")

if __name__ == '__main__':
    main()
