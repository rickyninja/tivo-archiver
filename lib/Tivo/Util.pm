package Tivo::Util;

use namespace::autoclean;
use Moose;
use TVrage;
use Data::Dumper;

has rage => (
    is => 'rw',
    isa => 'TVrage',
    default => sub {
        return TVrage->new;
    },
    lazy => 1,
);

sub get_filename {
    my $self = shift;
    my $video = shift || die 'missing details';

    my $detail = $video->video_details;
    my $filename = $detail->title;
    my $ep_num;

    if ($detail->is_episodic) {
        my $rage = $self->rage;
        my $rtv_show = eval { $rage->get_show($detail->title) } or do {
            my $e = $@;
            if ($e =~ /^Failed to match show in tvrage!/) {
                # Some shows have several candidates in the tvrage api, and no data
                # in the tivo to disambiguate the candidates (Being Human for example).
                # If the episode_number is all digits, it's hopefully accurate.
                $ep_num = join('x', $detail->episode_number =~ /^(\d{1,})(\d{2})$/);
            }
            else {
                die;
            }
        };

        my $s_ep;
        if ($rtv_show) {
            my $episodes = $rage->get_episodes($rtv_show->showid);
            $s_ep = $self->get_episode_tivo($detail, $episodes) || $ep_num || '0x00';
        }

        (my $episode_number = $s_ep) =~ s/x//;
        $detail->episode_number($episode_number);
        $filename .= " $s_ep-" . $detail->episode_title;
    }
    $filename =~ s{/}{-}g;   # filenames can't have slashes in them
    return $filename;
}

sub get_episode_tivo {
    my $self = shift;
    my $detail = shift || confess 'missing detail';
    my $episodes = shift || confess 'missing episodes';

    my $episode_title = $detail->episode_title;

    for (my $desperate = 0; $desperate <= 3; $desperate++) {
        for my $episode (@$episodes) {
            my $sea_num = $episode->season;
            my $rage_title = $episode->title;
            my $tivo_title = $episode_title;
            my $ep_num = $episode->seasonnum;

            my @rage_title = split //, $rage_title;
            my @tivo_title = split //, $tivo_title;

            # normalize chars ’ vs ' etc.
            for (my $i = 0; $i < $#rage_title; $i++) {
                my $chr = $rage_title[$i];
                my $ord = ord($chr);
                unless ($ord >= 32 && $ord <= 126) {
                    splice(@rage_title, $i, 1) if $i < $#rage_title;
                    splice(@tivo_title, $i, 1) if $i < $#tivo_title;
                }
            }
            $rage_title = lc(join '', @rage_title);
            $tivo_title = lc(join '', @tivo_title);

            # As we become more desperate to find a match strip out non-word characters
            # to make a match more likely.
            if ($desperate >= 2) {
                s/\W//g for ($tivo_title, $rage_title);
            }

            # exact title match
            if ($rage_title eq $tivo_title) {
                return join('x', $sea_num, $ep_num);
            }
            # match against production code (Charmed)
            elsif ($detail->episode_number && $episode->prodnum
                && $detail->episode_number eq $episode->prodnum
            ) {
                return join('x', $sea_num, $ep_num);
            }
            # exact title match if you add part_index inside parens to tivo title
            elsif ($detail->has_part_index && $desperate == 0) {
                my $tt = sprintf("$tivo_title (%d)", $detail->part_index);
                if ($tt eq $rage_title) {
                    return join('x', $sea_num, $ep_num);
                }
            }
            # rage title contains tivo title
            elsif ($desperate == 1 && $rage_title =~ /\Q$tivo_title\E/) {
                return join('x', $sea_num, $ep_num);
            }
            # tivo title contains rage title
            elsif ($desperate == 1 && $tivo_title =~ /\Q$rage_title\E/) {
                return join('x', $sea_num, $ep_num);
            }
            elsif ($desperate == 1) {
                # try to match 'Kill Billie: Vol.2' with 'Kill Billie (2)'
                if ($rage_title =~ /\(\d+\)/) {
                    my $rt = $rage_title;
                    $rt =~ s/\((\d+)\)//;
                    my $sequel = $1;
                    $rt =~ s/\s+$//;
                    if ($tivo_title =~ /\Q$rt\E/ && $tivo_title =~ /\Q$sequel\E/) {
                        return join('x', $sea_num, $ep_num);
                    }
                }
                elsif ($rage_title =~ / and / && $tivo_title =~ /&/) {
                    my $tt = $tivo_title;
                    $tt =~ s/&/and/g;
                    if ($rage_title eq $tt) {
                        return join('x', $sea_num, $ep_num);
                    }
                }
                elsif ($rage_title =~ /&/ && $tivo_title =~ / and /) {
                    my $tt = $tivo_title;
                    $tt =~ s/ and / & /g;
                    if ($rage_title eq $tt) {
                        return join('x', $sea_num, $ep_num);
                    }
                }
            }

        }
    }

    return;
}

sub get_pymeta {
    my $self = shift;
    my $video = shift || confess 'missing video';

    my $detail = $video->video_details;

    my @pymeta = (
        title         => $detail->title,
        seriestitle   => $detail->series_title,
        isEpisode     => ($detail->is_episode ? 'true' : 'false'),
        $detail->has_description ? (description => $detail->description) : (),
        $detail->has_time ? (time => $detail->time) : (),
    );

    for my $genre ($detail->series_genres) {
        push @pymeta, vProgramGenre => $genre;
    }

    for my $actor ($detail->actors) {
        push @pymeta, vActor => $actor;
    }

    for my $guest ($detail->guest_stars) {
        push @pymeta, vGuestStar => $guest;
    }

    for my $director ($detail->directors) {
        push @pymeta, vDirector => $director;
    }

    for my $exec ($detail->exec_producers) {
        push @pymeta, vExecProducer => $exec;
    }

    for my $prod ($detail->producers) {
        push @pymeta, vProducer => $prod;
    }

    for my $writer ($detail->writers) {
        push @pymeta, vWriter => $writer;
    }

    for my $host ($detail->hosts) {
        push @pymeta, vHost => $host;
    }

    for my $chore ($detail->choreographers) {
        push @pymeta, vChoreographer => $chore;
    }

    if ($detail->has_part_count && $detail->has_part_index) {
        push @pymeta,
            partCount => $detail->part_count,
            partIndex => $detail->part_index
        ;
    }

    if ($detail->is_episodic) {
        my $original_air_date = $detail->original_air_date;
        my $pi = sprintf("%02d", $detail->part_index || 0);
        $original_air_date =~ s/^(\d{4})-(\d{2})-(\d{2})T00:00:00Z$/$1-$2-$3T$pi:00:00Z/;

        push @pymeta,
            $detail->has_episode_title ? (episodeTitle => $detail->episode_title) : (),
            $detail->has_episode_number ? (episodeNumber => $detail->episode_number) :(),
            $detail->has_original_air_date ? (originalAirDate => $original_air_date) : (),
        ;
    }
    else {
        push @pymeta, movieYear => $detail->movie_year if $detail->has_movie_year;
    }

    my $pymeta = '';
    while (@pymeta) {
        my ($key, $val) = splice(@pymeta, 0, 2);
        $pymeta .= "$key: $val\n";
    }

    return $pymeta;
}


__PACKAGE__->meta->make_immutable;


1;

__END__

=head1 PURPOSE

Provide some utility methods for: correlating tivo metadata to ragetv metadata,
creating metadata files for pyTivo.

=head1 Obtaining accurate season and episode info

Tivo uses wonky values like the production codes as the episode_number; common for Charmed.
I've also seen tivo data have the season number without the episode number (4 vs. 402); common
for Angel and Buffy.  For these reasons the code will prefer ragetv season and episode data when
it can be matched up to the tivo title.

Ragetv alters multipart titles for some reason; I'm wondering if this is because ragetv data
is input by hand by it's user base.

 Examples
 tivo   - title: Buffy the Vampire Slayer, episodeTitle: Graduation Day,
          partIndex = 1, partCount = 2
 ragetv - title: Graduation Day (1)

 tivo   - title: Charmed, episodeTitle: Kill Billie: Vol.2
          partIndex = 2, partCount = 2
 ragetv - title: Kill Billie (2)

=head1 Author

Jeremy Singletary <jeremys@rickyninja.net>
