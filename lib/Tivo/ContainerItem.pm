package Tivo::ContainerItem;

use namespace::autoclean;
use Moose;

has in_progress => (
    is => 'rw',
    isa => 'Bool',
);

has content_type => (
    is => 'rw',
    isa => 'Str',
);

has source_format => (
    is => 'rw',
    isa => 'Str',
);

has title => (
    is => 'rw',
    isa => 'Str',
);

has source_size => (
    is => 'rw',
    isa => 'Int',
);

has duration => (
    is => 'rw',
    isa => 'Int',
);

has capture_date => (
    is => 'rw',
    isa => 'Int',
);

# This value will be in hex, convert to int on the fly.
# <CaptureDate>0x51ED10AE</CaptureDate>
around capture_date => sub {
    my $orig = shift;
    my $self = shift;
    if (@_) {
        return $self->$orig( hex shift );
    }
    else {
        return $self->$orig;
    }
};

has episode_title => (
    is => 'rw',
    isa => 'Str',
);

has description => (
    is => 'rw',
    isa => 'Str',
);

has source_channel => (
    is => 'rw',
    isa => 'Int',
);

has source_station => (
    is => 'rw',
    isa => 'Str',
);

has high_definition => (
    is => 'rw',
    isa => 'Str',
);

has program_id => (
    is => 'rw',
    isa => 'Str',
);

has series_id => (
    is => 'rw',
    isa => 'Str',
);

has episode_number => (
    is => 'rw',
    isa => 'Str',
);

has byte_offset => (
    is => 'rw',
    isa => 'Int',
);

has byte_offset => (
    is => 'rw',
    isa => 'Int',
);

has content_url => (
    is => 'rw',
    isa => 'Str',
);

has content_type_url => (
    is => 'rw',
    isa => 'Str',
);

has custom_icon_url => (
    is => 'rw',
    isa => 'Str',
);

has video_details_url => (
    is => 'rw',
    isa => 'Str',
);

has video_details => (
    is => 'rw',
    isa => 'Tivo::VideoDetails',
);

__PACKAGE__->meta->make_immutable;

1;

__END__

=head1 PURPOSE

Data class to hold tivo metadata about a container item.

=head1 Author

Jeremy Singletary <jeremys@rickyninja.net>
