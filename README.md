

## About

Stripe subscriptions can now have multiple plans per subscription. Many of you
have multiple subscriptions per customer. This tool will combine those
subscriptions into a single subscription. Now your customer will only be
charged once!

## Install

The easiest way to use `submigrate` is to download a pre-built binary.

- [OS X]()
- [Windows]()
- [Linux]()

You can also install using the Go tool.

```
go get github.com/stripe/submigrate
```

## Usage

`submigrate` takes a list of subscription IDs. `submigrate` is safe to run by
default; no actions will taken. You'll see logging explaining what would have
happened.

```
$ submigration -key <key> sub_9HmVS3IUUUVGYZ sub_9GbdjrGBMXCEaj
-----> Beginning subscription migration in <dryrun> mode

-----> Fetching requested subscriptions
       Getting subscription sub_9HmVS3IUUUVGYZ
       Getting subscription sub_9GbdjrGBMXCEaj
       Successfully fetched subscriptions

-----> Filtering out canceled subscriptions
       Ignoring canceled subscription sub_9HmVS3IUUUVGYZ

-----> Marking migration as complete
       Subscription sub_9GbdjrGBMXCEaj is active and all other subscriptions are canceled
```

If the proposed changes look reasonable, pass the `-run` flag to begin the
migration. `submigrate` is also idempotent; it's safe to run multiple times.

## Migration Strategy

In order to group existing subscriptions into one subscription, `submigrate`
will pick one subscription to be your primary subscription. Then, it will
cancel all the other subscriptions and add the plans of those subscriptions on
to the primary subscription.

Since the subscription will bill all items together, all the plans you are
grouping together must have the same interval and currency. Also, the plans
cannot have trial period days set.

When switching plans from one subscription to another, `submigrate` will
generate proration items for the difference between the end date of the
existing subscription and the end date of the primary subscription.

First, we pick the subscription with the maximum `current_period_end` to be the
primary subscription.

For each other subscription, add a new item to primary subscription with the same
plan and quantity as the existing subscription.

Lastly, cancel all subscriptions other than the primary subscription.
