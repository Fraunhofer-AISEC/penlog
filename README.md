# pypenlog

This package implements the [penlog specification](https://github.com/Fraunhofer-AISEC/penlog/blob/master/man/penlog.7.adoc) published by Fraunhofer AISEC.
pypenlog provides a `penlog` Python package which is available in [PyPi](https://pypi.org/project/pypenlog/).

## Quickstart

``` python 
import penlog

logger = penlog.Logger("testlogger")
logger.log_info("hi!")
```
