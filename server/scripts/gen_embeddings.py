"""Generate embeddings for knowledge chunks and store in pgvector."""
import json, requests, subprocess, sys

with open(r'C:\Users\29006\Desktop\opsmind\chunks.json', encoding='utf-8') as f:
    chunks = json.load(f)

texts = [c['content'] for c in chunks]
print(f'Processing {len(texts)} chunks...')

resp = requests.post('https://api.siliconflow.cn/v1/embeddings', json={
    'model': 'BAAI/bge-m3', 'input': texts, 'encoding_format': 'float'
}, headers={
    'Authorization': 'Bearer sk-pclemkqvcnntefyrtgehdxznpqazgpvqwwazznarfznfcprg',
    'Content-Type': 'application/json'
}, timeout=60)

if resp.status_code != 200:
    print(f'ERROR: {resp.status_code} {resp.text[:300]}')
    sys.exit(1)

data = resp.json()
emb = sorted(data['data'], key=lambda x: x['index'])
print(f'Got {len(emb)} embeddings, dim={len(emb[0]["embedding"])}')

# Build SQL and pipe to psql
sql_lines = []
for d in emb:
    chunk_id = chunks[d['index']]['id']
    vec = d['embedding']
    vec_str = '[' + ','.join(f'{v:.8f}' for v in vec) + ']'
    sql_lines.append(f"UPDATE knowledge_chunks SET embedding = '{vec_str}'::halfvec WHERE id = {chunk_id};")

sql = '\n'.join(sql_lines)
result = subprocess.run([
    'docker', 'exec', '-i', 'opsmind-postgres',
    'psql', '-U', 'opsmind', '-d', 'opsmind'
], input=sql, capture_output=True, text=True)

if result.returncode != 0:
    print(f'SQL Error: {result.stderr}')
    sys.exit(1)

print(f'Updated {len(sql_lines)} chunks with embeddings:')
for line in sql_lines[:3]:
    print(f'  {line[:80]}...')
