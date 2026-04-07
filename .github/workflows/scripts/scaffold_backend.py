#!/usr/bin/env python3
import os
import sys

feature = os.environ.get('FEATURE', '')
slug = os.environ.get('SLUG', '')
backend_files_str = os.environ.get('BACKEND_FILES', '')

print(f"=== Implementing: {feature} ===")

if not backend_files_str:
    print("No backend files to create")
    sys.exit(0)

files = [f.strip() for f in backend_files_str.split(';') if f.strip()]

for file_path in files:
    file_path = file_path.strip()
    if not file_path:
        continue

    os.makedirs(os.path.dirname(file_path), exist_ok=True)
    name = os.path.basename(file_path).rsplit('.', 1)[0]

    if file_path.endswith('.sql'):
        content = f"""-- Migration: {slug}
-- Created by: Ship Pipeline
-- TODO: Add migration SQL

CREATE TABLE IF NOT EXISTS {slug} (
id TEXT PRIMARY KEY,
created_at TIMESTAMPTZ DEFAULT NOW(),
updated_at TIMESTAMPTZ DEFAULT NOW()
);
"""
    else:
        content = f"""package {os.path.basename(os.path.dirname(file_path)) or 'main'}

import "context"

// {feature}
// TODO: Implement {name} handler
type {name.title().replace('_', '')} struct {{}}

func New{name.title().replace('_', '')}() *{name.title().replace('_', '')} {{
\treturn &{name.title().replace('_', '')}{{}}
}}

func (h *{name.title().replace('_', '')}) Handle(ctx context.Context) error {{
\t// TODO: Implement handler
\treturn nil
}}
"""

    with open(file_path, 'w') as f:
        f.write(content)
    print(f"Created: {file_path}")
