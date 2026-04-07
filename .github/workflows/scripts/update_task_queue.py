#!/usr/bin/env python3
import os
import subprocess
from datetime import date

today = date.today().isoformat()

# Get values from environment (set by GitHub Actions)
p0_count = os.environ.get('P0_COUNT', '0')
p1_count = os.environ.get('P1_COUNT', '0')
p2_count = os.environ.get('P2_COUNT', '0')
open_issues = os.environ.get('OPEN_ISSUES', '0')

# Find PRD files by priority
prds_dir = 'PRDs'
p0_prds, p1_prds, p2_prds = [], [], []

if os.path.isdir(prds_dir):
    for filename in os.listdir(prds_dir):
        if not filename.endswith('.md'):
            continue
        filepath = os.path.join(prds_dir, filename)
        try:
            with open(filepath, 'r') as f:
                content = f.read(500)
            if '[P0]' in content:
                p0_prds.append((filename, content))
            elif '[P1]' in content:
                p1_prds.append((filename, content))
            elif '[P2]' in content:
                p2_prds.append((filename, content))
        except Exception:
            pass

# Extract title from first heading
def get_title(content):
    for line in content.split('\n')[:5]:
        if line.startswith('# '):
            return line[2:].strip()
    return 'Unknown'

# Build output
lines = [
    f"# TASK_QUEUE.md - Hearth Development Queue\n",
    f"> Auto-updated by Competitive Analysis Pipeline\n",
    f"> Last update: {today}\n",
    f"\n",
    f"## Queue Statistics\n",
    f"- **P0 Features**: {p0_count}\n",
    f"- **P1 Features**: {p1_count}\n",
    f"- **P2 Features**: {p2_count}\n",
    f"- **Open GitHub Issues**: {open_issues}\n",
    f"\n",
    f"## Shipping Queue (Priority Order)\n",
    f"\n",
    f"### P0 - Critical (Must Ship)\n",
    f"\n",
]

for filename, content in p0_prds:
    title = get_title(content)
    base = filename[:-3]
    lines.append(f"- [ ] **P0** Feature: {title} -- {base}\n")

lines.extend([
    f"\n",
    f"### P1 - High Value (Next Sprint)\n",
    f"\n",
])

for filename, content in p1_prds:
    title = get_title(content)
    base = filename[:-3]
    lines.append(f"- [ ] **P1** Feature: {title} -- {base}\n")

lines.extend([
    f"\n",
    f"### P2 - Medium (Backlog)\n",
    f"\n",
])

for filename, content in p2_prds:
    title = get_title(content)
    base = filename[:-3]
    lines.append(f"- [ ] **P2** Feature: {title} -- {base}\n")

new_content = ''.join(lines)

# Check if changed
try:
    with open('TASK_QUEUE.md', 'r') as f:
        old_content = f.read()
except FileNotFoundError:
    old_content = ''

if new_content == old_content:
    print("TASK_QUEUE unchanged")
else:
    with open('TASK_QUEUE.md', 'w') as f:
        f.write(new_content)
    print("TASK_QUEUE updated")
    subprocess.run(['git', 'add', 'TASK_QUEUE.md'], check=True)
    subprocess.run(['git', 'config', 'user.email', 'greg@gregh.dev'], check=True)
    subprocess.run(['git', 'config', 'user.name', 'Greg Hendrickson'], check=True)
    subprocess.run(
        ['git', 'commit', '-m', f'chore: refresh TASK_QUEUE from competitive analysis ({today})'],
        check=True
    )
    subprocess.run(['git', 'push', 'origin', 'develop'], check=True)
