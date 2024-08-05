from fastapi import FastAPI, Request
import uvicorn
import os

app = FastAPI()

@app.get("/")
def read_root(request: Request):
      return request.headers

if __name__ == "__main__":
      host = os.getenv("HOST", "0.0.0.0")
      port = int(os.getenv("AWS_LWA_PORT", 8081))
      config = uvicorn.Config("main:app", host=host, port=port, log_level="info")
      server = uvicorn.Server(config)
      server.run()