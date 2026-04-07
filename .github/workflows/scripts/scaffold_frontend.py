#!/usr/bin/env python3
import os
import sys

feature = os.environ.get('FEATURE', '')
slug = os.environ.get('SLUG', '')
frontend_files_str = os.environ.get('FRONTEND_FILES', '')

print(f"=== Implementing frontend: {feature} ===")

if not frontend_files_str:
    print("No frontend files to create")
    sys.exit(0)

files = [f.strip() for f in frontend_files_str.split(';') if f.strip()]

for file_path in files:
    file_path = file_path.strip()
    if not file_path:
        continue

    os.makedirs(os.path.dirname(file_path), exist_ok=True)
    ext = os.path.splitext(file_path)[1].lstrip('.')

    if ext == 'svelte':
        content = f"""<script lang="ts">
/**
 * {feature}
 * TODO: Implement component
 */
const slug = "{slug}";
</script>

<div class="{slug}">
<p>{feature} - TODO: Implement</p>
</div>

<style>
.{slug} {{
/* TODO: Add styles */
}}
</style>
"""
    elif ext == 'ts':
        slug_name = slug.title().replace('-', '')
        content = f"""/**
 * {feature}
 * TODO: Implement store/logic
 */

export const {slug_name}Store = {{
// TODO: Implement
}};
"""
    else:
        content = f"""// {feature} - TODO: Implement {ext} file
"""

    with open(file_path, 'w') as f:
        f.write(content)
    print(f"Created: {file_path}")
