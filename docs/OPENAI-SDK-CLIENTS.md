# Clients OpenAI standard

Murmuration expose une API compatible OpenAI sur `/v1`. Les SDK OpenAI peuvent
donc parler au control plane en changeant seulement l'URL de base et le token.

Pour les SDK, l'URL de base doit inclure `/v1`, car les clients ajoutent ensuite
les chemins comme `/models` et `/chat/completions`.

```bash
export MURMURATION_BASE_URL=http://serveur:8080/v1
export MURMURATION_TOKEN=MON_TOKEN
```

Les exemples ci-dessous utilisent `llama3:8b`, comme dans le démarrage rapide.
Remplacez-le par un modèle renvoyé par `GET /v1/models`.

## Python

Installer le client :

```bash
python -m pip install openai
```

Lister les modèles puis faire une requête chat :

```python
import os

from openai import OpenAI


client = OpenAI(
    api_key=os.environ["MURMURATION_TOKEN"],
    base_url=os.environ.get("MURMURATION_BASE_URL", "http://serveur:8080/v1"),
)

models = client.models.list()
print([model.id for model in models.data])

response = client.chat.completions.create(
    model="llama3:8b",
    messages=[
        {"role": "user", "content": "Bonjour"},
    ],
)

print(response.choices[0].message.content)
```

Streaming SSE :

```python
stream = client.chat.completions.create(
    model="llama3:8b",
    messages=[
        {"role": "user", "content": "Réponds en une phrase."},
    ],
    stream=True,
)

for chunk in stream:
    delta = chunk.choices[0].delta.content
    if delta:
        print(delta, end="", flush=True)
print()
```

## JavaScript

Installer le client :

```bash
npm install openai
```

Créer `client.mjs` :

```js
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.MURMURATION_TOKEN,
  baseURL: process.env.MURMURATION_BASE_URL ?? "http://serveur:8080/v1",
});

const models = await client.models.list();
console.log(models.data.map((model) => model.id));

const response = await client.chat.completions.create({
  model: "llama3:8b",
  messages: [{ role: "user", content: "Bonjour" }],
});

console.log(response.choices[0].message.content);
```

Lancer :

```bash
node client.mjs
```

Streaming SSE :

```js
const stream = await client.chat.completions.create({
  model: "llama3:8b",
  messages: [{ role: "user", content: "Réponds en une phrase." }],
  stream: true,
});

for await (const chunk of stream) {
  const delta = chunk.choices[0]?.delta?.content;
  if (delta) {
    process.stdout.write(delta);
  }
}

process.stdout.write("\n");
```

## Contrat supporté aujourd'hui

La gateway implémente actuellement :

- `GET /v1/models`
- `POST /v1/chat/completions`
- `stream: true` sur `chat.completions`

Les autres routes OpenAI ne sont pas encore garanties par Murmuration.
