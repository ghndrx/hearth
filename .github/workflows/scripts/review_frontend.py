#!/usr/bin/env python3
import boto3
import subprocess

bedrock = boto3.client('bedrock-runtime', region_name='us-west-2')

try:
  diff = subprocess.check_output(
    ['git', 'diff', 'HEAD~10', '--', 'frontend/src/'],
    text=True
  )
except:
  diff = "No frontend changes"

prompt = (
  "Review Hearth frontend Svelte/SvelteKit code for: "
  "1) Accessibility, 2) Performance, 3) UX issues, 4) Code quality. "
  "Format as: ## Good / ## Issues / ## Suggestions.\n\n"
  f"Diff: {diff[:6000]}"
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
with open('frontend_review.md', 'w') as f:
  f.write(f"## Frontend AI Review\n\n{output}\n\n---\n*Generated via Kimi K2.5*\n")
