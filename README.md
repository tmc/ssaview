ssaview
-------
http://golang-ssaview.herokuapp.com/

ssaview is a small utlity that renders SSA code alongside input Go code

Runs via HTTP on :8080

License: ISC

```sh
  $ go get github.com/tmc/ssaview
  $ ssaview &
  $ open http://localhost:8080/
```

Deploying:

On Heroku: Uses custom buildpack to preserve GOROOT:

```sh
$ heroku buildpacks:set https://github.com/tmc/heroku-buildpack-go.gi
```

Screenshot:
![Example screenshot](https://github.com/tmc/ssaview/raw/master/.screenshot.png)
