# Robots.txt Traefik plugin

<!-- markdownlint-disable-next-line MD001 -->
#### Table of Contents

1. [Description](#description)
2. [Setup](#setup)
3. [Usage](#usage)
4. [Reference](#reference)
5. [Development](#development)
6. [Contributors](#contributors)

## Description

Robots.txt is a middleware plugin for [Traefik](https://traefik.io/) which add rules based on
[ai.robots.txt](https://github.com/ai-robots-txt/ai.robots.txt/) or on custom rules in `/robots.txt` of your website.

## Setup

### Configuration

```yaml
# Static configuration

experimental:
  plugins:
    example:
      moduleName: github.com/solution-libre/traefik-plugin-robots-txt
      version: v0.1.0
```

## Usage

```yaml
# Dynamic configuration

http:
  routers:
    my-router:
      rule: host(`localhost`)
      service: service-foo
      entryPoints:
        - web
      middlewares:
        - robots-txt

  services:
   service-foo:
      loadBalancer:
        servers:
          - url: http://127.0.0.1
  
  middlewares:
    robots-txt:
      plugin:
        traefik-plugin-robots-txt:
          aiRobotsTxt: true
```

## Reference

<!-- markdownlint-disable MD013 -->
| Name            | Description                                | Default value | Example                                  |
| --------------- | ------------------------------------------ | ------------- | ---------------------------------------- |
| aiRobotsTxt     | Enable the retrieval of ai.robots.txt list | `false`       | `true`                                   |
| additionalRules | Add additianl rules at the end of the file |               | `\nUser-agent: *\nDisallow: /private/\n` |
<!-- markdownlint-enable MD013 -->

## Development

[Solution Libre](https://www.solution-libre.fr)'s repositories are open projects,
and community contributions are essential for keeping them great.

[Fork this repo on GitHub](https://github.com/solution-libre/traefik-plugin-robots-txt/fork)

## Contributors

The list of contributors can be found at: <https://github.com/solution-libre/traefik-plugin-robots-txt/graphs/contributors>
