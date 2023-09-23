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

resource "render_service" "flask" {
  name   = "flask"
  repo   = "https://github.com/render-examples/flask-hello-world"
  type   = "web_service"
  branch = "master"

  web_service_details = {
    env    = "python"
    region = "frankfurt"
    plan   = "starter"
    native = {
      build_command = "pip install -r requirements.txt; echo 'hi'"
      start_command = "echo 'safe'; gunicorn app:app"
    }
  }
}

output "nextjs-url" {
  value = render_service.flask.web_service_details.url
}