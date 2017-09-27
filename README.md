# Heroku Buildpack for Pgpool-II

## Usage

This buildpack depends on [heroku-buildpack-apt](https://github.com/heroku/heroku-buildpack-apt).

You must configure the buildpacks in the following order:

```bash
heroku buildpacks:add https://github.com/sobrinho/heroku-buildpack-apt --index 1
heroku buildpacks:add https://github.com/sobrinho/heroku-buildpack-pgpool --index 2
heroku buildpacks:add https://github.com/sobrinho/heroku-buildpack-ruby --index 3
```

_heroku-buildpack-ruby is not a dependency, it's an example of an application buildpack which must be the latest, you may use whatever you want._

Remember that if your application is already deployed, you may not need the 3rd command as your application buildpack will be already set.

## Followers

Once you have created a follower on Heroku as the [documentation](https://devcenter.heroku.com/articles/heroku-postgres-follower-databases):

```
$ heroku pg:info
=== DATABASE_URL, HEROKU_POSTGRESQL_CHARCOAL_URL
Plan:        Standard 0
Status:      available
...

$ heroku addons:create heroku-postgresql:standard-2 --follow HEROKU_POSTGRESQL_CHARCOAL_URL
Adding heroku-postgresql:standard-2 to sushi... done, v71 ($200/mo)
Attached as HEROKU_POSTGRESQL_WHITE
Follower will become available for read-only queries when up-to-date
Use `heroku pg:wait` to track status

$ heroku pg:wait
Waiting for database HEROKU_POSTGRESQL_WHITE_URL... available
```

You can configure the pgpool buildpack by using the `PGPOOL_URLS`.

This variable expects to have the `HEROKU_POSTGRESQL_*COLOR*_URL` names, not the scheme itself, example:

```
heroku config:set PGPOOL_URLS='HEROKU_POSTGRESQL_CHARCOAL_URL HEROKU_POSTGRESQL_WHITE'
```

The first one must be the master database and the others the followers.

## Using external databases

If you are using external databases like [Amazon RDS](https://aws.amazon.com/rds/) or similar instead of [Heroku Postgres](https://www.heroku.com/postgres), you need to configure each database on its own variable and then configure them on `PGPOOL_URLS`, i.e.:

```
heroku config:set MASTER_URL=postgres://user:password@ip:port/dbname FOLLOWER_URL=postgres://user:password@ip:port/dbname PGPOOL_URLS='MASTER_URL FOLLOWER_URL'
```

If you have a server of your own, it's probably better to configure the pgpool on that server instead of the Heroku itself.

Be aware that there is pros and cons of using the pgpool at the database server instead of the application server, do some research on that at [Stack Overflow](https://stackoverflow.com) and [Server Fault](https://serverfault.com) to understand this better.

## Load Balancing

This buildpack configures pgpool as load balancing and sets the databases automatically on `pgpool.conf`.

If you have any questions, check the [Pgpool's documentation](http://www.pgpool.net) first and/or ask at [Stack Overflow](https://stackoverflow.com) or [Server Fault](https://serverfault.com).

In the case you understand shell scripting and pgpool configuration, you can take a look at [bin/start-pgpool](bin/start-pgpool) and [etc/pgpool.conf](etc/pgpool.conf).

## Application

You need to create an Aptfile with the following contents:

```
pgpool2
```

This will download and install the [pgpool](http://www.pgpool.net) on [Heroku](https://heroku.com) using the [heroku-buildpack-apt](https://github.com/heroku/heroku-buildpack-apt).

Configure your `Procfile` to call bin/start-pgpool before the application itself, example:

```
web: bin/start-pgpool bundle exec puma -p $PORT
```

This will start the pgpool on `127.0.0.1:5432` and call `bundle exec puma -p $PORT`.

If you have other dyno types that you want to use the pgpool, you need to do the same:

```
web: bin/start-pgpool bundle exec puma -p $PORT
worker: bin/start-pgpool bundle exec sidekiq
```

This command will do all the job and configure the `DATABASE_URL` to point to the pgpool instance.

## Running commands on Heroku

Be in mind that if you call any command on Heroku, like `heroku run console`, it won't connect to the pgpool and instead will connect directly to the master database.

Whenever you want the application to connect to the pgpool, you need to prepend the `bin/start-pgpool`, like this:

```
heroku run bin/start-pgpool bundle exec console
heroku run bin/start-pgpool bundle exec rake something
```

If you get you doing this a lot, you can override the console and rake commands in your Procfile, or similar if you aren't using [Rails](http://rubyonrails.org), like this:

```
web: bin/start-pgpool bundle exec puma -p $PORT
worker: bin/start-pgpool bundle exec sidekiq
console: bin/start-pgpool bundle exec rails console
rake: bin/start-pgpool bundle exec rake
```

## Enable/Disable

You can temporarily disable pgpool by using the `PGPOOL_ENABLED`, example:

```
heroku config:set PGPOOL_ENABLED=0 # disables pgpool
heroku config:set PGPOOL_ENABLED=1 # enables pgpool
```

## FAQ

### This will start one pgpool per dyno?

Yes. If you have 10 web dynos, you will have 10 pgpools running at the same time, each dyno with its own.

### This won't be an issue?

It shouldn't as pgpool is a proxy that sits between the application and the database.

You have to understand that the application sees the pgpool as a regular postgres database and the postgres database sees the pgpool as a regular client.

There is nothing special in that.

### What if I wan't to use the replication feature of pgpool?

To be honest, I have no idea what will happen with multiple pgpool running in that case.

Feel free to make a pull request answering that here.

### I have other questions

As always, take a look at [Pgpool's documentation](http://www.pgpool.net), [Stack Overflow](https:/stackoverflow.com), [Server Fault](https://serverfault.com) and [Google](https://www.google.com) before opening an issue.

## TODO

I will be happy to accept pull requests addressing these points:

  1. We need automated tests on this buildpack
  2. Remove the dependency of heroku-buildpack-apt and have the pgpool binary here instead
  3. As we accomplish the previous point, we need an automated way to upgrade/downgrade the pgpool binary
