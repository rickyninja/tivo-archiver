package TVrage;

use namespace::autoclean;
use Moose;
use LWP;
use XML::LibXML;
use TVrage::Show;
use TVrage::Episode;
use Cache::File;
use URI::Escape;

has debug => (is => 'rw', default => sub { $ENV{DEBUG} || 0 });
has lwp => (
    is => 'rw',
    isa => 'LWP::UserAgent',
    default => sub {
        my $self = shift;
        my $lwp = LWP::UserAgent->new;
        if ($self->debug) {
            $lwp->add_handler(request_send => sub { shift->dump; return });
            $lwp->add_handler(response_done => sub { shift->dump; return });
        }
        return $lwp;
    },
    lazy => 1,
);

has base_uri => (
    is => 'rw',
    default => 'http://services.tvrage.com',
);

has region => (
    is => 'rw',
    isa => 'Str',
);

# I initially set the cache to never expire.  I don't think the code
# would be able to get episodes that aired after the cache was stored unless
# the cache is allowed to expire.
has cache => (
    is => 'rw',
    default => sub {
        my $cache = Cache::File->new(
            cache_root => '/tmp/tvrage-cache',
            default_expires => '1 week',
        );
        return $cache;
    },
    lazy => 1,
);


around 'go' => sub {
    my $orig = shift;
    my $self = shift;
    my ($route) = @_;

    my $xml = $self->cache->get($route);
    unless ($xml) {
        $xml = $self->$orig(@_);
        $self->cache->set($route, $xml);
    }
    return $xml;
};

around 'get_episodes' => sub {
    my $orig = shift;
    my $self = shift;

    my $xml = $self->$orig(@_);
    return $self->get_episodes_obj_from_xml($xml);
};

around 'get_show' => sub {
    my $orig = shift;
    my $self = shift;

    my $xml = $self->$orig(@_);
    my $show_name = $_[0];

    my $parser = XML::LibXML->new;
    my $doc = $parser->load_xml( string => $xml );

    my @shows;
    for my $item ($doc->findnodes('/Results/show')) {
        my $name = $item->findvalue('name');
        my $country = $item->findvalue('country');
        my $arg = {
            showid         => $item->findvalue('showid'),
            name           => $name,
            link           => $item->findvalue('link'),
            country        => $country,
            started        => $item->findvalue('started'),
            ended          => $item->findvalue('ended'),
            seasons        => $item->findvalue('seasons'),
            status         => $item->findvalue('status'),
            classification => $item->findvalue('classification'),
            genres => [ map { $_->to_literal } $item->findnodes('genres/genre') ],
        };
        my $show = TVrage::Show->new($arg);
        push @shows, $show;
    }

    return @shows;
};

sub get_episodes_obj_from_xml {
    my $self = shift;
    my $xml = shift || confess 'missing xml';

    my $parser = XML::LibXML->new;
    my $doc = $parser->load_xml( string => $xml );

    my @episodes;
    for my $item ($doc->findnodes('/Show/Episodelist')) {
        for my $season ($item->findnodes('Season')) {
            for my $ep ($season->findnodes('episode')) {
                my $arg = {
                    season    => $season->findvalue('@no'),
                    epnum     => $ep->findvalue('epnum'),
                    seasonnum => $ep->findvalue('seasonnum'),
                    prodnum   => $ep->findvalue('prodnum'),
                    airdate   => $ep->findvalue('airdate'),
                    link      => $ep->findvalue('link'),
                    title     => $ep->findvalue('title'),
                };
                my $episode = TVrage::Episode->new($arg);
                push @episodes, $episode;
            }
        }
    }

    return wantarray ? @episodes : \@episodes;
}

sub find_show {
    my $self = shift;
    my $show_name = shift;

    my $region = $self->region;
    my @shows = $self->get_show($show_name);

    # Lost Girl is listed as CA country in tvrage, and so my configured region of US
    # will not match.  Lost Girl has only aired in CA though, so you can get a string
    # equality match on the 2nd $retry (the old behavior).
    for (my $retry = 0; $retry <= 1; $retry++) {
        for my $show (@shows) {
            my $name = $show->name;
            my $country = $show->country;

            if ($region && $retry < 1) {
                # substring match because $show_name could look like "Wilfred" or "Wilfred (US)"
                if ($region eq $country && $name =~ /^\Q$show_name\E/) {
                    return $show;
                }
            }
            # Attempt an exact title match if $region is not set or $retry > 0.
            else {
                if ($name eq $show_name) {
                    return $show;
                }
            }
        }
    }

    confess 'Failed to match show in tvrage!';
}

sub get_show {
    my $self = shift;
    my $show = shift || confess 'missing show';

    $show = uri_escape($show);
    my $route = "/feeds/search.php?show=$show";
    return $self->go($route);
}

sub get_episodes {
    my $self = shift;
    my $show_id = shift || confess 'missing show_id';

    $show_id = uri_escape($show_id);
    my $route = "/feeds/episode_list.php?sid=$show_id";
    return $self->go($route);
}

sub go {
    my $self = shift;
    my $route = shift || confess 'missing route';

    my $uri = $self->base_uri . $route;
    my $r = $self->lwp->get($uri);
    confess $r->status_line if $r->is_error;
    return $r->content;
}


__PACKAGE__->meta->make_immutable;


1;

__END__

=head1 PURPOSE

Client to communicate with tvrage service.

=head1 Author

Jeremy Singletary <jeremys@rickyninja.net>
