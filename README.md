# go-indexer

## Motivation

This is a refactoring of [NAME]'s [NAME] code to do two things:

1. To be able to index and search multiple `gocloud.dev/blob` instances. By default that just means mulitple directories on the same filesystem but technically it means that anything which supports the `gocloud.dev/blob.Bucket` interface could be indexed.
2. To be able to export and import search "archives" derived from earlier indexings.

That's it. All the "hard" stuff is all still [NAME]'s original code.

## Tools

```
$> make cli
go build -mod vendor -ldflags="-s -w" -o bin/index cmd/index/main.go
go build -mod vendor -ldflags="-s -w" -o bin/search cmd/search/main.go
```

### index

```
$> ./bin/index -bucket-uri cwd:// -index-uri cwd:///index.idx
$> du -h index.idx 
1.2M	index.idx
``

### search

```
$> ./bin/search -bucket-uri cwd:// 
enter search term: 
aaronland
--------------
9 index result(s)

&{.git/config 0}
9. 	url = git@github.com:aaronland/go-indexer.git

&{bucket.go 0}
9. 	"github.com/aaronland/gocloud-blob/bucket"
13. // START OF put me in aaronland/gocloud-blob
47. // END OF put me in aaronland/gocloud-blob

&{cmd/index/main.go 0}
8. 	"github.com/aaronland/go-indexer"

&{cmd/search/main.go 0}
10. 	"github.com/aaronland/go-indexer"

&{go.mod 0}
1. module github.com/aaronland/go-indexer
8. 	github.com/aaronland/gocloud-blob v0.0.17

&{go.sum 0}
13. github.com/aaronland/gocloud-blob v0.0.17 h1:TjsM6uT+XQ8SejlFNDgyxOXKEc90gZlPI0ov2EcMUHI=
14. github.com/aaronland/gocloud-blob v0.0.17/go.mod h1:Mk/2NKSaWsLTTwdqE3AEVms4W5v+Wv1WS1Z5HyZmhHA=

&{index.go 0}
16. 	"github.com/aaronland/gocloud-blob/bucket"
17. 	"github.com/aaronland/gocloud-blob/walk"

&{vendor/github.com/whosonfirst/go-ioutil/readseekcloser.go 0}
4. // (20210217/thisisaaronland)

&{vendor/modules.txt 0}
1. # github.com/aaronland/gocloud-blob v0.0.17
3. github.com/aaronland/gocloud-blob/bucket
4. github.com/aaronland/gocloud-blob/walk

enter search term: 
```

It is also possible to load an existing index to query. For example:

```
$> ./bin/search -index-uri cwd:///index.idx
enter search term: 
sfomuseum
--------------
7 index result(s)

&{cmd/index/main.go 0}
9. 	"github.com/sfomuseum/go-flags/multi"

&{cmd/search/main.go 0}
11. 	"github.com/sfomuseum/go-flags/multi"

&{go.mod 0}
9. 	github.com/sfomuseum/go-flags v0.10.0

&{vendor/modules.txt 0}
18. # github.com/sfomuseum/go-flags v0.10.0
20. github.com/sfomuseum/go-flags/multi

enter search term:
```

_Note: In the example above results from indexing the `.git` folder were excluded._

## Things this package doesn't do (yet)

* There is no way to exclude certain files from being indexed yet so be careful what you choose to index.

## See also

* https://github.com/google/go-cloud
* https://github.com/aaronland/gocloud-blob
