package TVrage::Episode;

use namespace::autoclean;
use Moose;

# This doesn't reset to 1 at the start of each season.
has epnum => (
    is => 'rw',
    isa => 'Int',
);

# This is the episode number for the current season.
# This one resets to 1 each season.
has seasonnum => (
    is => 'rw',
    isa => 'Int',
);

# This isn't a tvrage attribute, but I'm adding it as a convenience.
# This attribute is what I expected the seasonnum to be.
has season => (
    is => 'rw',
    isa => 'Int',
);

# This is the production code.
# It's poorly named since it can contain non-numbers.
has prodnum => (
    is => 'rw',
    isa => 'Str',
);

has airdate => (
    is => 'rw',
    isa => 'Str', # yyyy-mm-dd
);

# This is a link to the tvrage site with info on the episode.
has link => (
    is => 'rw',
    isa => 'Str',
);

# This is the episode's title.
has title => (
    is => 'rw',
    isa => 'Str',
);

__PACKAGE__->meta->make_immutable;

1;

__END__

=head1 PURPOSE

Data class to hold tvrage metadata about an episode.  This class isn't strictly necessary,
but serves to better explain some of the tvrage episode attributes.

=head1 Author

Jeremy Singletary <jeremys@rickyninja.net>
