import yaml

with open("/home/sanju/ai-project/ai_gateway/backend/api/openapi/openapi.yaml", "r") as f:
    spec = yaml.safe_load(f)

# Add schemas
if "components" not in spec:
    spec["components"] = {"schemas": {}}

schemas = spec["components"]["schemas"]
schemas["Policy"] = {
    "type": "object",
    "properties": {
        "id": {"type": "string", "format": "uuid"},
        "organization_id": {"type": "string", "format": "uuid"},
        "name": {"type": "string"},
        "description": {"type": "string"},
        "enforcement": {"type": "string", "enum": ["active", "disabled", "permissive"]},
        "priority": {"type": "integer"},
        "created_at": {"type": "string", "format": "date-time"},
        "updated_at": {"type": "string", "format": "date-time"},
    }
}
schemas["PolicyRule"] = {
    "type": "object",
    "properties": {
        "id": {"type": "string", "format": "uuid"},
        "policy_id": {"type": "string", "format": "uuid"},
        "organization_id": {"type": "string", "format": "uuid"},
        "rule_type": {"type": "string"},
        "description": {"type": "string"},
        "conditions": {"type": "object", "additionalProperties": True},
        "parameters": {"type": "object", "additionalProperties": True},
        "enabled": {"type": "boolean"},
        "created_at": {"type": "string", "format": "date-time"},
    }
}

# Add requests
schemas["CreatePolicyRequest"] = {
    "type": "object",
    "required": ["name", "enforcement", "priority"],
    "properties": {
        "name": {"type": "string"},
        "description": {"type": "string"},
        "enforcement": {"type": "string", "enum": ["active", "disabled", "permissive"]},
        "priority": {"type": "integer"},
    }
}
schemas["UpdatePolicyRequest"] = {
    "type": "object",
    "properties": {
        "name": {"type": "string"},
        "description": {"type": "string"},
        "enforcement": {"type": "string", "enum": ["active", "disabled", "permissive"]},
        "priority": {"type": "integer"},
    }
}
schemas["CreatePolicyRuleRequest"] = {
    "type": "object",
    "required": ["rule_type", "enabled"],
    "properties": {
        "rule_type": {"type": "string"},
        "description": {"type": "string"},
        "conditions": {"type": "object", "additionalProperties": True},
        "parameters": {"type": "object", "additionalProperties": True},
        "enabled": {"type": "boolean"},
    }
}
schemas["UpdatePolicyRuleRequest"] = {
    "type": "object",
    "properties": {
        "description": {"type": "string"},
        "conditions": {"type": "object", "additionalProperties": True},
        "parameters": {"type": "object", "additionalProperties": True},
        "enabled": {"type": "boolean"},
    }
}
schemas["PolicyList"] = {
    "type": "object",
    "properties": {
        "items": {
            "type": "array",
            "items": {"$ref": "#/components/schemas/Policy"}
        }
    }
}
schemas["PolicyRuleList"] = {
    "type": "object",
    "properties": {
        "items": {
            "type": "array",
            "items": {"$ref": "#/components/schemas/PolicyRule"}
        }
    }
}

# Add paths
paths = spec.setdefault("paths", {})
paths["/v1/policies"] = {
    "get": {
        "tags": ["policies"],
        "operationId": "listPolicies",
        "responses": {
            "200": {
                "description": "OK",
                "content": {"application/json": {"schema": {"$ref": "#/components/schemas/PolicyList"}}}
            }
        }
    },
    "post": {
        "tags": ["policies"],
        "operationId": "createPolicy",
        "requestBody": {
            "required": True,
            "content": {"application/json": {"schema": {"$ref": "#/components/schemas/CreatePolicyRequest"}}}
        },
        "responses": {
            "201": {
                "description": "Created",
                "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Policy"}}}
            }
        }
    }
}
paths["/v1/policies/{id}"] = {
    "parameters": [{"name": "id", "in": "path", "required": True, "schema": {"type": "string", "format": "uuid"}}],
    "get": {
        "tags": ["policies"],
        "operationId": "getPolicy",
        "responses": {
            "200": {
                "description": "OK",
                "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Policy"}}}
            }
        }
    },
    "patch": {
        "tags": ["policies"],
        "operationId": "updatePolicy",
        "requestBody": {
            "required": True,
            "content": {"application/json": {"schema": {"$ref": "#/components/schemas/UpdatePolicyRequest"}}}
        },
        "responses": {
            "200": {
                "description": "OK",
                "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Policy"}}}
            }
        }
    },
    "delete": {
        "tags": ["policies"],
        "operationId": "deletePolicy",
        "responses": {
            "204": {"description": "No Content"}
        }
    }
}

paths["/v1/policies/{id}/rules"] = {
    "parameters": [{"name": "id", "in": "path", "required": True, "schema": {"type": "string", "format": "uuid"}}],
    "get": {
        "tags": ["policies"],
        "operationId": "listPolicyRules",
        "responses": {
            "200": {
                "description": "OK",
                "content": {"application/json": {"schema": {"$ref": "#/components/schemas/PolicyRuleList"}}}
            }
        }
    },
    "post": {
        "tags": ["policies"],
        "operationId": "createPolicyRule",
        "requestBody": {
            "required": True,
            "content": {"application/json": {"schema": {"$ref": "#/components/schemas/CreatePolicyRuleRequest"}}}
        },
        "responses": {
            "201": {
                "description": "Created",
                "content": {"application/json": {"schema": {"$ref": "#/components/schemas/PolicyRule"}}}
            }
        }
    }
}

paths["/v1/policies/{id}/rules/{ruleId}"] = {
    "parameters": [
        {"name": "id", "in": "path", "required": True, "schema": {"type": "string", "format": "uuid"}},
        {"name": "ruleId", "in": "path", "required": True, "schema": {"type": "string", "format": "uuid"}}
    ],
    "patch": {
        "tags": ["policies"],
        "operationId": "updatePolicyRule",
        "requestBody": {
            "required": True,
            "content": {"application/json": {"schema": {"$ref": "#/components/schemas/UpdatePolicyRuleRequest"}}}
        },
        "responses": {
            "200": {
                "description": "OK",
                "content": {"application/json": {"schema": {"$ref": "#/components/schemas/PolicyRule"}}}
            }
        }
    },
    "delete": {
        "tags": ["policies"],
        "operationId": "deletePolicyRule",
        "responses": {
            "204": {"description": "No Content"}
        }
    }
}

paths["/v1/policies/preview"] = {
    "post": {
        "tags": ["policies"],
        "operationId": "previewPolicy",
        "responses": {
            "200": {
                "description": "OK",
                "content": {"application/json": {"schema": {"type": "object", "additionalProperties": True}}}
            }
        }
    }
}

paths["/v1/policies/{id}/validate"] = {
    "parameters": [{"name": "id", "in": "path", "required": True, "schema": {"type": "string", "format": "uuid"}}],
    "post": {
        "tags": ["policies"],
        "operationId": "validatePolicy",
        "responses": {
            "200": {
                "description": "OK",
                "content": {"application/json": {"schema": {"type": "object", "additionalProperties": True}}}
            }
        }
    }
}

with open("/home/sanju/ai-project/ai_gateway/backend/api/openapi/openapi.yaml", "w") as f:
    yaml.dump(spec, f, sort_keys=False)
