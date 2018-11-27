# v0.5.2

* upd: support any logging package with a `Printf` method via `Logger` interface rather than forcing `log.Logger` from standard log package
* upd: remove explicit log level classifications from logging messages
* upd: switch to errors package (for `errors.Wrap` et al.)
* upd: clarify error messages
* upd: refactor tests
* fix: `SearchCheckBundles` to use `*SearchFilterType` as its second argument
* fix: remove `NewAlert` - not applicable, alerts are not created via the API
* add: ensure all `Delete*ByCID` methods have CID corrections so short CIDs can be passed

# v0.5.1

* upd: retryablehttp to start using versions that are now available instead of tracking master

# v0.5.0

* Initial - promoted from github.com/circonus-labs/circonus-gometrics/api to an independant package
