from typing import Any

def handler(event: dict[str, Any], context: dict[str, Any]) -> dict[str, Any]:
    return { "echo": f"{event['msg']}"}

