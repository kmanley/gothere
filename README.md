# gothere

gothere is a simple HTTP redirector. It loads a text file with mappings from short URLs to
long URLs. When it gets a request for a short URL, it redirects to the long URL.
 
## Why don't you just use bit.ly?

The namespace for bit.ly short URLs is shared across *all* custom domains. Every time I 
tried to customize a URL I got the dreaded 'This Custom Bitlink is already taken' error.
That made me sad.

## Quick start

Edit urls.txt to specify your mappings. 
* One URL per line
* Short URLs must start with /
* Separate the short URL and long URL with at least one space
* Comment lines and empty lines are ignored

```bash
# this is a Comment

/g http://google.com
/m https://www.gmail.com
/a/short/url http://some/really/long/url
```

Then just run gothere to start your redirector

## Options

You can optionally specify the listening port number, and the default URL to
redirect to if the requested URL is not in the map.

```bash
Usage of ./gothere:
  -defaultUrl="http://google.com": default URL
  -port=80: listening port
```

## Updating the URL mapping

To update the URL mapping without downtime, edit urls.txt and send SIGHUP. 
This causes gothere to reload the mappings while continuing to serve HTTP.

## How do I daemonize it?

I recommend using the excellent [supervisord](http://supervisord.org). Here's an 
example program block for /etc/supervisor/supervisord.conf

```bash
[program:gothere]
command=/go/src/github.com/kmanley/gothere/gothere -port=80 -defaultUrl=http://whatever.com
directory=/go/src/github.com/kmanley/gothere
stdout_logfile=/var/log/gothere-stdout
stderr_logfile=/var/log/gothere-stderr
```

### License

MIT License
