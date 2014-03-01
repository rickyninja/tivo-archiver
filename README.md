# PURPOSE

Download tivo recordings marked as 'keep until I delete' and organize them on a media server.
Intended to be a complement to pyTivo.

# SYNOPSIS

I run it via cron.

    jeremys@jeremys-desktop> cat /etc/cron.d/tivo-archiver 
    @reboot root touch /var/run/tivo-archiver.pid && chown jeremys:jeremys /var/run/tivo-archiver.pid

    jeremys@jeremys-desktop> crontab -l | grep tivo-archiver
    \*/10 \* \* \* \* /home/jeremys/bin/tivo-archiver > /dev/null 2>&1

# Author

Jeremy Singletary <jeremys@rickyninja.net>

# Compatible tivo devices

I wrote this to work with my series 2 tivo.  It has not been tested on any of the others.

# Wishlist

- remove commercials

I've done this a few times by hand using Avidemux, which is works well and is simple enough.
It's tedious however, and I don't want to do it by hand for every recording I archive.  It
results in significant space savings on your drive though, so it'd be a nice bonus.  It looked
like ffmpeg would be able to identify black frames for me, but I wasn't able to get it to work.

- delete recordings from tivo

It'd be nice to be able to delete a recording from the tivo via HMO protocol after a
successful download.

# Known Problems

- Tivo network failure during consecutive downloads

I have a series 2 tivo, and it will drop off the network after many consecutive downloads.
I'm assuming this is a network driver problem.  The tivo still shows it's IP address info as
though it were connected, but I no longer see my pyTivo server, and the tivo is not pingable.
The tivo will recover after a restart.

While troubleshooting this I removed my tivo branded wireless usb adapter, and my tivo
unexpectedly restarted; so you probably don't want to do that while recording is in progress.

To mitigate this issue, I inserted a 10 minute sleep after each download.  The problem persists,
but is greatly diminished.

- Downloads are slow

This is a limitation of the cpu on series 2 tivos.  These units weren't designed for transferring
lots of content.  From what I've read, the max throughput is 4 Mbs wired or wireless.

In normal use this doesn't really bother me.  It's most annoying when I'm coding a new feature
and I want the downloads to go faster so I can determine if my code changes work as expected.

Due to this limitation I record at High quality, and not Best quality.  I tried Best and then
the transfer speed became an annoyance.  High quality recording of 1 hour of content weighs in
at about 1.6 GB, where Best quality weighs in around 2.6 GB.

- Same show, different region

Being Human has US and UK versions, and I don't see anything in the tivo data to match the region.
To work around this problem, you can configure your region in /etc/tivo-archiver.yml.  The value for 
this attribute is used to match the country attribute in the tvrage api.

If a show already has correct episode number in tivo data (instead of the production code
or incorrect data), this will be used as a last resort as it tends to be less reliable.