from fastapi import FastAPI, Request

app = FastAPI(title="example")

@app.middleware("http")
async def proxy_aware_swagger(request: Request, call_next):
        forwarded_for_prefix = request.headers.get("x-forwarded-prefix", "")
        if forwarded_for_prefix != "":
            request.scope["root_path"] = forwarded_for_prefix
        response = await call_next(request)
        return response

@app.get("/")
def read_root(request: Request):
      return request.headers