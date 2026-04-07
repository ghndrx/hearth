#!/usr/bin/env python3
import boto3
import subprocess

bedrock = boto3.client('bedrock-runtime', region_name='us-west-2')

try:
  commits = subprocess.check_output(
    ['git', 'log', '--oneline', '-10', 'origin/develop'],
    text=True
  )
except:
  commits = "Unable to fetch"

try:
  diff = subprocess.check_output(
    ['git', 'diff', 'HEAD~10', '--', 'backend/'],
    text=True
  )
except:
  diff = "No backend changes"

prompt = (
  "Review Hearth backend Go code changes for: "
  "1) Security issues, 2) Performance, 3) Code quality, 4) Best practices. "
  "Format as: ## Good / ## Issues / ## Suggestions / ## Security Alerts.\n\n"
  f"Commits: {commits[:1000]}\nDiff: {diff[:6000]}"
)

messages = [{"role": "user", "content": [{"text": prompt}]}]

try:
  response = bedrock.converse(
    modelId='moonshotai.kimi-k2.5',
    messages=messages,
    inferenceConfig={"maxTokens": 2048, "temperature": 0.3}
  )
  output = response['output']['message']['content'][0]['text']
except Exception as e:
  output = f"Review skipped: {e}"

print(output)
with open('backend_review.md', 'w') as f:
  f.write(f"## Backend AI Review\n\n{output}\n\n---\n*Generated via Kimi K2.5*\n")
