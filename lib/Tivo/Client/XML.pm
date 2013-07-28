package Tivo::Client::XML;

use Moose::Role;
use XML::LibXML;
use Tivo::ContainerItem;
use Tivo::VideoDetails;

around [qw(query_item)] => sub {
    my $orig = shift;
    my $self = shift;
    
    my $doc = $self->remove_bad_xmlns( $self->$orig(@_) );
    #print $doc->toString; exit;

    my %detail;
    for my $item ($doc->findnodes('/TiVoItem/Item/Details/*')) {
        $detail{ $item->nodeName } = $item->to_literal;
    }

    return \%detail;
};


around [qw(get_details)] => sub {
    my $orig = shift;
    my $self = shift;

    my $doc = $self->remove_bad_xmlns( $self->$orig(@_) );
    my $detail = Tivo::VideoDetails->new;

    my $is_episode = $doc->findvalue('//showing/program/isEpisode');
    $detail->is_episode($is_episode) if $is_episode;

    my $is_episodic =  $doc->findvalue('//showing/program/series/isEpisodic');
    $detail->is_episodic($is_episodic) if $is_episodic;

    my $title = $doc->findvalue('//showing/program/title');
    $detail->title($title) if $title;

    my $series_title = $doc->findvalue('//showing/program/series/seriesTitle') || $detail->title;
    $detail->series_title($series_title) if $series_title;

    my $description = $doc->findvalue('//showing/program/description');
    if ($description) {
        $description =~ s/\s*Copyright Tribune Media Services, Inc\.$//;
        $detail->description($description)
    }

    # don't see seriesId anywhere in xml

    if ($detail->is_episodic) {
        $detail->time('OAD');
        my $oad = $doc->findvalue('//showing/program/originalAirDate');
        $detail->original_air_date($oad) if $oad;

        # episodeNumber is frequently incorrect in tivo metadata,
        # prefer the value acquired from tvrage (implemented in Tivo::Util).
        my $episode_number = $doc->findvalue('//showing/program/episodeNumber');
        $detail->episode_number($episode_number) if $episode_number;

        my $episode_title = $doc->findvalue('//showing/program/episodeTitle');
        $detail->episode_title($episode_title) if $episode_title;
    }
    else {
        my $time = $doc->findvalue('//showing/time');
        $detail->time($time) if $time;

        my $movie_year = $doc->findvalue('//showing/program/movieYear');
        $detail->movie_year($movie_year) if $movie_year;
    }

    my $part_count = $doc->findvalue('//showing/partCount');
    $detail->part_count($part_count) if $part_count;

    my $part_index = $doc->findvalue('//showing/partIndex');
    $detail->part_index($part_index) if $part_index;

    my @multis = (
        {
            method => 'series_genres',
            xpath => '//showing/program/series/vSeriesGenre/element',
        },
        {
            method => 'actors',
            xpath => '//showing/program/vActor/element',
        },
        {
            method => 'guest_stars',
            xpath => '//showing/program/vGuestStar/element',
        },
        {
            method => 'directors',
            xpath => '//showing/program/vDirector/element',
        },
        {
            method => 'exec_producers',
            xpath => '//showing/program/vExecProducer/element',
        },
        {
            method => 'producers',
            xpath => '//showing/program/vProducer/element',
        },
        {
            method => 'choreographers',
            xpath => '//showing/program/vChoreographer/element',
        },
        {
            method => 'writers',
            xpath => '//showing/program/vWriter/element',
        },
        {
            method => 'hosts',
            xpath => '//showing/program/vHost/element',
        },
    );

    for my $multi (@multis) {
        my $method = $multi->{method};
        my $xpath = $multi->{xpath};
        my $values = [
            map { $_->to_literal } $doc->findnodes($xpath)
        ];
        $detail->$method($values) if @$values >= 1;
    }

    return $detail;
};
    
around [qw(query_container)] => sub {
    my $orig = shift;
    my $self = shift;
    
    my $doc = $self->remove_bad_xmlns( $self->$orig(@_) );
    my @items;
    for my $item ($doc->findnodes('/TiVoContainer/Item')) {
        my $content_type = $item->findvalue('Details/ContentType');
        next if $content_type eq 'x-tivo-container/folder';

        my $ci = Tivo::ContainerItem->new;

        my $in_progress = $item->findvalue('Details/InProgress');
        $ci->in_progress(1) if lc($in_progress) eq 'yes';

        $ci->content_type($content_type) if $content_type;

        my $source_format = $item->findvalue('Details/SourceFormat');
        $ci->source_format($source_format) if $source_format;

        my $title = $item->findvalue('Details/Title');
        $ci->title($title) if $title;

        my $source_size = $item->findvalue('Details/SourceSize');
        $ci->source_size($source_size) if $source_size;

        my $duration = $item->findvalue('Details/Duration');
        $ci->duration($duration) if $duration;

        my $capture_date = $item->findvalue('Details/CaptureDate');
        $ci->capture_date($capture_date) if $capture_date;

        my $episode_title = $item->findvalue('Details/EpisodeTitle');
        $ci->episode_title($episode_title) if $episode_title;

        my $description = $item->findvalue('Details/Description');
        $ci->description($description) if $description;

        my $source_channel = $item->findvalue('Details/SourceChannel');
        $ci->source_channel($source_channel) if $source_channel;

        my $source_station = $item->findvalue('Details/SourceStation');
        $ci->source_station($source_station) if $source_station;

        my $high_definition = $item->findvalue('Details/HighDefinition');
        $ci->high_definition($high_definition) if $high_definition;

        my $program_id = $item->findvalue('Details/ProgramId');
        $ci->program_id($program_id) if $program_id;

        my $series_id = $item->findvalue('Details/SeriesId');
        $ci->series_id($series_id) if $series_id;

        my $episode_number = $item->findvalue('Details/EpisodeNumber');
        $ci->episode_number($episode_number) if $episode_number;

        my $byte_offset = $item->findvalue('Details/ByteOffset');
        $ci->byte_offset($byte_offset);

        my $content_url = $item->findvalue('Links/Content/Url');
        $ci->content_url($content_url) if $content_url;

        my $content_type_url = $item->findvalue('Links/Content/ContentType');
        $ci->content_type_url($content_type_url) if $content_type_url;

        my $custom_icon_url = $item->findvalue('Links/CustomIcon/Url');
        $ci->custom_icon_url($custom_icon_url) if $custom_icon_url;

        my $video_details_url = $item->findvalue('Links/TiVoVideoDetails/Url');
        $ci->video_details_url($video_details_url) if $video_details_url;

        push @items, $ci;
    }
    return \@items;
};

# XML::LibXML doesn't work unless the xmlns attribute is removed.
sub remove_bad_xmlns {
    my $self = shift;
    my $xml = shift || confess 'missing xml!';

    $xml =~ s{xmlns=["'][^"']+["']}{}g;
    my $parser = XML::LibXML->new;
    my $doc = $parser->load_xml( string => $xml );
    return $doc;
}


1;

__END__

=head1 PURPOSE

Convert XML returned from Tivo into objects with equivalent data.

=head1 Author

Jeremy Singletary <jeremys@rickyninja.net>
