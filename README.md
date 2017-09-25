# Heroku Buildpack for Pgpool-II

Configure your Procfile to call bin/start-pgpool before the application, i.e.:

```
web: bin/start-pgpool bundle exec rackup -p $PORT
```

This will start the pgpool on 127.0.0.1:5432 and call `bundle exec rackup -p $PORT`.
