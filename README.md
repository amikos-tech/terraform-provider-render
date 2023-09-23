# Render Provider

The Render provider is used to interact with https://render.com

Use the navigation to the left to read about the available resources.

## Supported Service Types

* Web Service
* Private Service
* Static Site

## Example Usage

Do not keep your authentication password in HCL for production environments, use Terraform environment variables.

**Setup environment**

```terraform
terraform {
  required_providers {
    render = {
      version = "1.1.1"
      source  = "render.com/terraform/render"
    }
  }
}

provider "render" {
  # api_key = "<YOUR_API_KEY>"    // Uses env.RENDER_API_KEY, if not supplied
  # email = "<YOUR_RENDER_EMAIL>" // Uses env.RENDER_EMAIL, if not supplied
}

resource "render_service" "client" {
  name = "client"
  repo = "https://github.com/render-examples/nextjs-hello-world"
  type = "static_site"

  static_site_details = {
    build_command = "yarn; yarn build; yarn next export"
    publish_path  = "out"
  }
}

resource "render_service" "api" {
  name = "api"
  repo = "https://github.com/render-examples/hapi-quick-start"
  type = "web_service"

  web_service_details = {
    env = "node"

    native = {
      build_command = "npm install"
      start_command = "node server.js"
    }
  }
}

resource "render_service" "db" {
  name = "db"
  repo = "https://github.com/render-examples/mongodb"
  type = "private_service"

  private_service_details = {
    env  = "docker"
    disk = {
      name       = "db"
      mount_path = "/data/db"
      size_gb    = 10
    }
  }
}

resource "render_service_environment" "api" {
  service = render_service.api.id

  variables = [{
      key   = "DATABASE_URL"
      value = render_service.db.private_service_details.url
  }]
}


resource "render_service_environment" "client" {
  service = render_service.client.id

  variables = [{
      key   = "API_URL"
      value = render_service.api.web_service_details.url
  }]
}
```

**Create web / private / static services**

```terraform
provider "render" {
  apiKey = "your-api-key"
  email  = "your-render-email"
}

resource "render_service" "nextjs" {
  name   = "nextjs"
  repo   = "https://github.com/render-examples/nextjs-hello-world"
  type   = "web_service"
  branch = "master"

  web_service_details = {
    env    = "node"
    region = "frankfurt"
    plan   = "starter"
    native = {
      build_command = "yarn; yarn build"
      start_command = "yarn start"
    }
  }
}


resource "render_service" "python_app" {
  name   = "my_python_app"
  repo   = "https://github.com/render-examples/flask-hello-world"
  type   = "web_service"
  branch = "master"

  web_service_details = {
    env    = "python"
    region = "frankfurt"
    plan   = "starter"
    native = {
      build_command = "pip install -r requirements.txt"
      start_command = "gunicorn app:app"
    }
    disk = {
      name = "my_app_data"
      mount_path = "/my_app_data"
      size_gb = 10
    }
  }
}

resource "render_service" "mongodb" {
  name = "mongodb"
  repo = "https://github.com/render-examples/mongodb"
  type = "private_service"

  private_service_details = {
    env = "docker"

    disk = {
      name       = "db"
      mount_path = "/data/db"
      size_gb    = 10
    }
  }
}

resource "render_service" "svelte" {
  name = "mongodb"
  repo = "https://github.com/render-examples/svelte"
  type = "static_site"

  static_site_details = {
    build_command = "npm install && npm run build"
    publish_path  = "public"
  }
}
```

## Render API Documentation

Here is a link to the official Render API documentation: https://api-docs.render.com/reference/introduction
