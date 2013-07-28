package Tivo::VideoDetails;

use namespace::autoclean;
use Moose;

has is_episode => (
    is => 'rw',
    isa => 'Bool',
    predicate => 'has_is_episode',
);

has is_episodic => (
    is => 'rw',
    isa => 'Bool',
    predicate => 'has_is_episodic',
);

around [ qw(is_episode is_episodic) ] => sub {
    my $orig = shift;
    my $self = shift;

    if (@_) {
        my $value = shift;
        $value = 1 if $value eq 'true';
        $value = 0 if $value eq 'false';
        return $self->$orig($value);
    }

    return $self->$orig;
};

has title => (
    is => 'rw',
    isa => 'Str',
    predicate => 'has_title',
);

has series_title => (
    is => 'rw',
    isa => 'Str',
    predicate => 'has_series_title',
);

has episode_title => (
    is => 'rw',
    isa => 'Str',
    predicate => 'has_episode_title',
);

# Typed as a string because tivo uses production codes sometimes, which aren't all digits.
has episode_number => (
    is => 'rw',
    isa => 'Str',
    predicate => 'has_episode_number',
);

has description => (
    is => 'rw',
    isa => 'Str',
    predicate => 'has_description',
);

has original_air_date => (
    is => 'rw',
    isa => 'Str',
    predicate => 'has_original_air_date',
);

has time => (
    is => 'rw',
    isa => 'Str',
    predicate => 'has_time',
);

has part_count => (
    is => 'rw',
    isa => 'Int',
    predicate => 'has_part_count',
);

has part_index => (
    is => 'rw',
    isa => 'Int',
    predicate => 'has_part_index',
);

has series_genres => (
    is => 'rw',
    isa => 'ArrayRef',
    auto_deref => 1,
    predicate => 'has_series_genres',
);

has actors => (
    is => 'rw',
    isa => 'ArrayRef',
    auto_deref => 1,
    predicate => 'has_actors',
);

has guest_stars => (
    is => 'rw',
    isa => 'ArrayRef',
    auto_deref => 1,
    predicate => 'has_guest_stars',
);

has directors => (
    is => 'rw',
    isa => 'ArrayRef',
    auto_deref => 1,
    predicate => 'has_directors',
);

has exec_producers => (
    is => 'rw',
    isa => 'ArrayRef',
    auto_deref => 1,
    predicate => 'has_exec_producers',
);

has producers => (
    is => 'rw',
    isa => 'ArrayRef',
    auto_deref => 1,
    predicate => 'has_producers',
);

has writers => (
    is => 'rw',
    isa => 'ArrayRef',
    auto_deref => 1,
    predicate => 'has_writers',
);

has hosts => (
    is => 'rw',
    isa => 'ArrayRef',
    auto_deref => 1,
    predicate => 'has_hosts',
);

has choreographers => (
    is => 'rw',
    isa => 'ArrayRef',
    auto_deref => 1,
    predicate => 'has_choreographers',
);

has movie_year => (
    is => 'rw',
    isa => 'Int',
    predicate => 'has_movie_year',
);



__PACKAGE__->meta->make_immutable;

1;

__END__

=head1 PURPOSE

Data class to hold tivo metadata about recording details. 

=head1 Author

Jeremy Singletary <jeremys@rickyninja.net>
