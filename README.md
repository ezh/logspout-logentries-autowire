# logspout-logentries-autowire

Logentries adaptor for logspout, using container labels to route logs.

Logs from containers without a `logentries.token` label will be ignored. Set this label to the token of the Logentries endpoint you want to use. eg 'aba74900-65b1-4fcb-9096-bbecdd3fc04c'

# Usage

* Logspout must reference the module

```
package main

import (
    _ "github.com/mergermarket/logspout-logentries-autowire"
)
```

* Logspout must be started with `/bin/logspout  logentriesautowire://`


## Developing

    # start the development container
    ./dev.sh
    
    # initialise the environment
    /logspout-logentries-autowire/init.sh
    
    # build and run logspout with the adapter
    /logspout-logentries-autowire/run.sh