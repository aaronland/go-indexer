# go-indexer

Bloom filter based search index with support for persistent archives.

## Motivation

This is a refactoring of Ben Boyter's [indexer](https://github.com/boyter/indexer) code to do two things:

1. To be able to index and search multiple `gocloud.dev/blob` instances. By default that just means mulitple directories on the same filesystem but technically it means that anything which supports the `gocloud.dev/blob.Bucket` interface could be indexed.
2. To be able to export and import search "archives" derived from earlier indexings.

That's it. All the "hard" stuff is all still Ben's original code.

This is meant to be a simple tool for indexing  arbitrary text, like free-form notes or a directory full of [Who's On First documents](https://github.com/whosonfirst-data) and providing a good-enough-is-good-enough interface for querying those files.

## Tools

```
$> make cli
go build -mod vendor -ldflags="-s -w" -o bin/index cmd/index/main.go
go build -mod vendor -ldflags="-s -w" -o bin/search cmd/search/main.go
```

### index

```
$> ./bin/index -h
Usage of ./bin/index:
  -bucket-uri value
    	One or more valid gocloud.dev/blob bucket URIs to index. The URI 'cwd://` will be interpreted as the current working directory on the local disk.
  -index-uri string
    	A valid gocloud.dev/blob bucket URIs containing the filename of the index to archive. (default "cwd:///indexer.idx")
```

For example:

```
$> ./bin/index -bucket-uri cwd:// -index-uri cwd:///index.idx
$> du -h index.idx 
1.2M	index.idx
```

### search

```
$> ./bin/search -h
Usage of ./bin/search:
  -bucket-uri value
    	One or more valid gocloud.dev/blob bucket URIs to index. The URI 'cwd://` will be interpreted as the current working directory on the local disk.
  -index-uri string
    	An optional valid gocloud.dev/blob bucket URIs containing the filename of the index (archive) to load (instead of indexing things from scratch). The URI scheme 'cwd://' will be interpreted as the current working directory on the local disk.
```

For example:

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
* It does not do incremental updates to existing indices.
* It does not remove individual items from existing indices.
* Probably none of the other things you'd like it to do.

## See also

* https://github.com/boyter/indexer
* https://github.com/google/go-cloud
* https://github.com/aaronland/gocloud-blob