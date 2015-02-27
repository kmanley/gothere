# gothere

gothere is a simple HTTP redirector. It loads a text file with mappings from short URLs to
long URLs. When it gets a request for a short URL, it redirects to the long URL.
 
## Why don't you just use bit.ly?

The namespace for bit.ly short URLs is shared across *all* custom domains. Every time I 
tried to customize a URL I got the dreaded 'This Custom Bitlink is already taken' error.
That made me sad.

## Quick start

Edit **urls.txt** (in the same directory as the binary) to specify your mappings. 
* One URL per line
* Short URLs must start with /
* Long URLs must start with protocol://
* Separate the short URL and long URL with at least one space
* Comment lines and empty lines are ignored
* In the case of duplicate short URLs the last one wins

```bash
# this is a Comment

/g http://google.com
/m https://www.gmail.com
/a/short/url http://some/really/long/url
```

Then just run gothere to start your redirector

## Options

You can specify the listening port number via the **-port** option, 
and the default URL to redirect to if the requested URL is not in the map
via the **-defaultUrl** option. The rest of the options are for the 
excellent [glog](https://github.com/golang/glog) logging library.

```bash
$ ./gothere -h
Usage of ./gothere:
  -alsologtostderr=false: log to standard error as well as files
  -defaultUrl="http://google.com": default URL
  -log_backtrace_at=:0: when logging hits line file:N, emit a stack trace
  -log_dir="": If non-empty, write log files in this directory
  -logtostderr=false: log to standard error instead of files
  -port=80: listening port
  -stderrthreshold=0: logs at or above this threshold go to stderr
  -v=0: log level for V logs
  -vmodule=: comma-separated list of pattern=N settings for file-filtered logging
```

## Updating the URL mapping

To update the URL mapping without downtime, edit urls.txt and send SIGHUP. 
This causes gothere to reload the mappings while continuing to serve HTTP.

## How do I daemonize it?

I recommend using the excellent [supervisord](http://supervisord.org). Here's an 
example program block for /etc/supervisor/supervisord.conf. You can either let 
glog log directly to files, or redirect glog to stderr and let supervisord
handle the log files. 

```bash
[program:gothere]
command=/go/src/github.com/kmanley/gothere/gothere -logtostderr=true -port=80 -defaultUrl=http://whatever.com
directory=/go/src/github.com/kmanley/gothere
stdout_logfile=/var/log/gothere-stdout
stderr_logfile=/var/log/gothere-stderr
```

### Production use
gothere currently powers [joyrd.link](http://joyrd.link), the short domain for my
wife's indoor cycling studio.

### License

MIT License
