package TVrage::Show;

use namespace::autoclean;
use Moose;

has showid => (
    is => 'rw',
    isa => 'Int',
);

has name => (
    is => 'rw',
    isa => 'Str',
);

has link => (
    is => 'rw',
    isa => 'Str',
);

has country => (
    is => 'rw',
    isa => 'Str',
);

has started => (
    is => 'rw',
    isa => 'Int',
);

has ended => (
    is => 'rw',
    isa => 'Int',
);

has seasons => (
    is => 'rw',
    isa => 'Int',
);

has status => (
    is => 'rw',
    isa => 'Str',
);

has classification => (
    is => 'rw',
    isa => 'Str',
);

has genres => (
    is => 'rw',
    isa => 'ArrayRef',
);

__PACKAGE__->meta->make_immutable;

1;

__END__

=head1 PURPOSE

Data class to hold tvrage metadata about a show.

=head1 Author

Jeremy Singletary <jeremys@rickyninja.net>
