workflow "Mirror Workflow" {
  on = "push"
  resolves = ["Mirror Action"]
}

action "Mirror Action" {
  uses = "spyoungtech/mirror-action@master"
  secrets = ["GIT_PASSWORD"]
  args = "https://gitlab.com/markus-wa/demoinfocs-golang.git"
  env = {
    GIT_USERNAME = "markus-wa"
  }
}
