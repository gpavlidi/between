# between

### What

A transparent Http/Https proxy made in Go that intercepts and optionaly modifies outgoing requests/ incoming responses. Currently supports only OSX since it uses [pf](https://developer.apple.com/library/mac/documentation/Darwin/Reference/ManPages/man5/pf.conf.5.html) for intercepting traffic.

### Why

While developing or reverse engineering web services, it's very useful to be able to see the incoming/outgoing traffic and have the ability to edit both. It's even more useful if you can programmaticaly alter them via code and do all that transparently to the 2 sides of communication. Currently and to my knowledge there isn't a library/software that does ALL of the above, so I decided to make one.

### How

[between](#between) uses [pf](https://developer.apple.com/library/mac/documentation/Darwin/Reference/ManPages/man5/pf.conf.5.html) to intercept all incoming and outgoing traffic. It will then filter out http and https requests and responses and will run user defined functions to edit them in any way possible. When [between](#between) exits it deactivates pf and restores network connectivity.

### Examples

- [deface](examples/deface.go) is a fun little demo app that sniffs out all incoming images, performs face detection, and replaces faces with the 'rage guy'. See how it looks below: :-p
<img width="970" alt="linkedin_defaced" src="https://cloud.githubusercontent.com/assets/1408921/10708038/7a5bc116-79b0-11e5-88f4-ab015ff5f502.png">
To run it :
```
$ go build examples/deface.go
$ sudo ./examples/deface
```

### Limitations

Currently [between](#between) has 2 limitations:
- Needs to run as root. This is for 2 reasons:
	- To be able to manipulate pf so it's intercepting all traffic. A different user can be used if permissions on pfctl are set.
	- To exclude requests made from [between](#between) from being intercepted again. Right now there is a pf rule that prevents traffic from root from being intercepted. A different user could be used for this reason too.
- Works only in OSX since it's using pf magic for interception. FreeBSD might work too but hasnt been tested.

### Alternatives

There are many alternatives that achieve some of [between](#between)'s functionality but none was satisfying everything mentioned on the [Why section](###Why). 
- Browser Plugins ([Chrome DevTools](https://developer.chrome.com/devtools), [Firebug](http://getfirebug.com/), [TamperData](https://addons.mozilla.org/en-US/firefox/addon/tamper-data/) etc): Works great for reviewing requests/responses but editing them is impossible/very limited.
- [Charles Proxy](http://www.charlesproxy.com/): Great proxy with ton of functionality. It's difficult (impossible?) to programmaticaly modify requests/responses in an arbitrary way.
