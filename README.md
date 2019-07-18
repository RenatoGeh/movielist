# Pop, the Corn

![pop](https://raw.githubusercontent.com/RenatoGeh/movielist/master/pop.png)

## What in the world is this?

A Telegram bot. For managing your and your friend's to-watch movie lists.

## How to I download the code?

You can either do it the Go way:

```
go get -u github.com/RenatoGeh/movielist
cd $GOPATH/src/github.com/RenatoGeh/movielist
```

Or the local way:

```
git clone https://github.com/RenatoGeh/movielist
cd movielist
```

## How do I make it do things?

Create your own bot from @BotFather. Copy your new bot's token and write
it to a file `token.tk`. Once you're done, compile:

```
go build
```

And you're ready to run.

```
./movielist
```

## How do I boss my bot around?

Try `/help`.
