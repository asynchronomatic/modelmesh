curl -s  http://localhost:10080/api/ps  -H "Content-Type: application/json" | jq .

curl  http://localhost:10080/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen3.8:latest",
    "stream": false,
    "messages": [
      {
        "role": "user",
        "content": "Hi Friend!"
      }
    ]
  }' 
