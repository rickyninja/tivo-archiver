#!/usr/bin/perl

use strict;
use warnings;
use Test::More tests => 1;
use Data::Dumper;
use Tivo::Client;
use Tivo::Util;
use TVrage;
use Local::TestInput qw(get_tivo_xml get_tvrage_xml);

my $tivo = Tivo::Client->new;
my $util = Tivo::Util->new;
my $rage = TVrage->new;

my $tivo_xml = get_tivo_xml('smallville', 'vortex');
my $tvrage_xml = get_tvrage_xml('smallville');

my $detail = $tivo->get_detail_obj_from_xml($tivo_xml);
my $episodes = $rage->get_episodes_obj_from_xml($tvrage_xml);
my $season_by_episode = $util->get_episode_tivo($detail, $episodes) || '';

# tivo title:   'Vortex'
# tvrage title: 'Vortex (2)'
ok($season_by_episode eq '2x01', 'tvrage title contains tivo title');
