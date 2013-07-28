package Tivo::Client;

use namespace::autoclean;
use Moose;
use Moose::Util::TypeConstraints;
use LWP;
use URI::Escape;
use Cache::File;

with 'Tivo::Client::XML';

has use_cache => (
    is => 'rw',
    isa => 'Bool',
    default => 1,
);

has _cache => (
    is => 'rw',
    default => sub {
        my $cache = Cache::File->new(
            cache_root => '/tmp/tivo-cache',
            default_expires => '3600 sec',
        );
        return $cache;
    },
    lazy => 1,
);

has _debug => (
    is => 'rw',
    isa => 'Bool',
    default => sub { $ENV{DEBUG} || 0 },
);

has lwp => (
    is => 'ro',
    isa => 'LWP::UserAgent',
    builder => '_lwp_builder',
    lazy => 1,
);

has _base_uri => (
    is => 'ro',
    isa => 'Str',
    builder => '_base_uri_builder',
    lazy => 1,
);

has tivo_protocol => (
    is => 'rw',
    isa => 'Str',
    default => 'https',
    lazy => 1,
);

has tivo_host => (
    is => 'rw',
    isa => 'Str',
);

has _realm => (
    is => 'rw',
    isa => 'Str',
    default => 'TiVo DVR',
    lazy => 1,
);

has tivo_login => (
    is => 'rw',
    isa => 'Str',
    default => 'tivo',
    lazy => 1,
);

has mak => (
    is => 'rw',
    isa => 'Int',
);

sub query_container {
    my $self = shift;
    my $param = shift;

    $param->{Command} = 'QueryContainer';
    return $self->go($param);
}

sub query_item {
    my $self = shift;
    my $param = shift || confess 'Missing param!';

    $param->{Command} = 'QueryItem';
    unless ($param->{Url}) {
        confess 'Url is required with the QueryItem command!';
    }
    return $self->go($param);
}

sub query_server {
    my $self = shift;
    my $param = shift;

    $param->{Command} = 'QueryServer';
    return $self->go($param);
}

sub reset_server {
    my $self = shift;
    my $param = shift;

    $param->{Command} = 'ResetServer';
    return $self->go($param);
}

sub query_formats {
    my $self = shift;
    my $param = shift;

    unless ($param->{SourceFormat}) {
        confess "SourceFormat is required with the QueryContainer command! It should be a valid mime type.";
    }
    $param->{Command} = 'QueryFormats';
    return $self->go($param);
}

sub get_details {
    my $self = shift;
    my $uri = shift || confess 'missing uri';

    if ($self->use_cache) {
        my $cached = $self->_cache->get($uri);
        return $cached if $cached;
    }

    my $request = HTTP::Request->new(GET => $uri);
    my $response = $self->lwp->request($request);
    if ($response->is_error) {
        confess $response->request->as_string . $response->as_string;
    }

    my $content = $response->decoded_content;
    $self->_cache->set($uri, $content);
    return $content;
}

sub go {
    my $self = shift;
    my $param = shift;

    my $lwp = $self->lwp;
    my $uri = $self->add_query( $self->_base_uri, $param );

    if ($self->use_cache) {
        my $cached = $self->_cache->get($uri);
        return $cached if $cached;
    }

    my $request = HTTP::Request->new( GET => $uri );
    my $response = $lwp->request( $request );
    if ($response->is_error) {
        confess $response->request->as_string . $response->as_string;
    }

    my $content = $response->decoded_content;
    $self->_cache->set($uri, $content);
    return $content;
}

sub add_query {
    my $self = shift;
    my $uri = shift || confess 'missing uri!';
    my $param = shift;

    my $querystring = '';

    while (my ($k,$v) = each %$param) {
        $v = uri_escape( $v );
        if ($querystring && substr($querystring, -1, 1) ne '&') {
            $querystring .= '&';
        }
        $querystring .= "$k=$v";
    }
    $querystring ? return "$uri?$querystring" : return $uri;
}

sub _base_uri_builder {
    my $self = shift;

    my $uri = sprintf("%s://%s/TiVoConnect", $self->tivo_protocol, $self->tivo_host);
    return $uri;
}

sub _lwp_builder {
    my $self = shift;

    my $lwp = LWP::UserAgent->new( 'ssl_opts' => { 'verify_hostname' => 0 } );
    $lwp->agent( __PACKAGE__ );

    $lwp->cookie_jar( {} ); # Without cookies, you'll get http 400 bad request for downloads.

    if ($self->_debug) {
        $lwp->add_handler( request_send => sub { shift->dump; return } );
        $lwp->add_handler( response_done => sub { shift->dump; return } );
    }

    for my $port (80, 443) {
        $lwp->credentials( join(':', $self->tivo_host, $port), $self->_realm, $self->tivo_login, $self->mak );
    }
    return $lwp;
}


__PACKAGE__->meta->make_immutable;

1;

__END__

=head1 PURPOSE

Client to communicate with Tivo via Calypso protocol (HMO).

=head1 Author

Jeremy Singletary <jeremys@rickyninja.net>
