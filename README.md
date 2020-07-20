# upduck
Upduck is a simple HTTP and HTTPs static file server that integrates with [DuckDNS](https://www.duckdns.org/). It is meant for quickly sharing static files beyond your local network over an HTTPs connection but can also be used within your local network (without HTTPs).

**Disclaimer**: This project has no affiliation with DuckDNS or Let's Encrypt and is not endorsed or supported by them.

**Disclaimer**: Using this program with the `-email` flag signifies your acceptance to the [Let's Encrypt's Subscriber Agreement and/or Terms of Service](https://letsencrypt.org/repository/).

### How to use
The help section of the program tries to be as helpful as possible:

```
$ upduck -h
upduck, a simple HTTP and HTTPs file server

Command-line flags:
  -dir string
    	Directory that should be served (default ".")
  -disallow-listings
    	Don't show directory listings
  -email string
    	Email sent to LetsEncrypt for certificate registration
  -p int
    	HTTP server port (default 8080)
  -save
    	Save the given command line arguments to a config file located in your home directory
  -site string
    	Your duckdns.org subdomain name, e.g. "test" for test.duckdns.org
  -sp int
    	HTTPS server port (default 443)
  -token string
    	The token you get from duckdns.org

Examples:
  Start a simple HTTP server on the default port:

    > upduck 

  Start a simple HTTP server on port 2020 that doesn't show directory listings:

    > upduck -p 2020 -disallow-listings

  Serve files from a specific directory (default is working directory):

    upduck -dir path/to/dir

  Start a HTTP server and a HTTPs server:

    > upduck -email your@email.com -token DuckDNSToken -site mysite

    For this one, your router must forward any incoming connection on a port of your choosing to port 443 (or the one set with the -sp option) of the device upduck runs on. 
    This external chosen port you set in the router must be put after the DuckDNS URL, e.g. https://mysite.duckdns.org:525/ for port 525.
    If you're not sure about how this works, search for "port forward tutorial" and your router model/vendor.

  Start a HTTP server and a HTTPs server on custom ports:

    > upduck -p 2020 -sp 2121 -email your@email.com -token DuckDNSToken -site mysite

    Here, the above notice also applies - ports (in this case 2121) must be forwarded in your router.

  You can also save your configuration so you don't need to type out everything all the time. Just run it normal and add the -save flag:

    > upduck -save -p 2020 -email your@email.com -token DuckDNSToken -site mysite

    This will save your current command line. The next time, you will just need to run upduck without arguments to start the same configuration.
    You can also add the -p flag without impacting your HTTPs configuration.
```

### Install
You can either compile this program or download a release.

#### Downloading
You can [download a version for your system](https://github.com/xarantolus/upduck/releases/latest) and move it anywhere you want. It is recommended to put the executable in a directory from your [`$PATH`](https://superuser.com/a/284351).

On a Raspberry Pi you would copy the link and start downloading the program:

    wget https://github.com/.../upduck-raspberrypi

Mark it as executable:

    chmod +x upduck-raspberrypi

Then move it to your `$PATH` to make it accessible everywhere:

    mv upduck-raspberrypi /usr/bin/upduck

Now you should be able to run `upduck` from anywhere. This is especially useful combined with using `-dir .` when [saving settings](#saving-settings) as it will serve the current directory you're in.

If you want to use ports below `1024` and run `upduck` without root (sudo), you can [set the `CAP_NET_BIND_SERVICE` permission](https://stackoverflow.com/a/414258):

    setcap 'cap_net_bind_service=+ep' /usr/bin/upduck

#### Compiling
Since this is a normal Go program, compiling works like this:

    go build

If you're compiling for another operating system, you can set environment variables. You can see the [`release.sh`](release.sh) script to see how it's done for building releases.

### Obtaining a DuckDNS domain and setting up HTTPs
To get a DuckDNS subdomain, you'll need to register [on their site](https://www.duckdns.org) and then [create a domain](https://www.duckdns.org/domains). The prefix you type in is the `-site` parameter of your program, your token is for the `-token` option.

Now that we have the domain, we'll need to make sure the router is set up correctly. For this, you'll need to forward a port in your router to port `443` (or the one set with `-sp`) of the device `upduck` is running on. This port will be part of your external address, e.g. `mysite.duckdns.org:port`, where `port` is a number.

When you did that, you can run `upduck` like this:

    upduck -email your@email.com -token DuckDNSToken -site mysite

The email address will be sent to [Let's Encrypt ](https://letsencrypt.org/) as part of obtaining an HTTPs certificate.

This should start a local HTTP web server on port `8080` and an HTTPs server on port `443`. The second one should receive the requests that are forwarded from your router.

### Saving settings
Since typing out all arguments can become tiresome, you can save them quite easily. They will then be reloaded on the next start.

**Disclaimer**: Writing your credentials to disk is a security risk since other programs might read that file. They could gain control over your DuckDNS account or find out your email address.

To save settings, just add the `-save` flag to your normal command line. The next time, your credentials will be restored and you don't have to remember them or type them every time.

### Loading settings
Saved settings are loaded automatically if no new options for DuckDNS are given.

When the config file is loaded, the following settings can be overwritten by command line flags: port with `-p` and directory listings with `-disallow-listings`. This means that you can run `upduck -p 2020` to get the local server while *still* getting the DuckDNS server if it was ever set up with `-save`.

### Contributions
Contributions, suggestions, questions and any issue reports are very welcome. Please don't hesistate to ask :)

### [License](LICENSE)
This is free as in freedom software. Do whatever you like with it.