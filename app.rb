require 'bundler'
Bundler.require

$LOAD_PATH.unshift File.join(File.dirname(__FILE__), 'lib')

require 'active_support/core_ext/object/try'
require 'active_support/core_ext/hash/keys'

require 'bits_service'

BitsService::Environment.init

require 'bits_service/helpers/config'
helpers BitsService::Helpers::Config

set :dump_errors, false if ENV['RACK_ENV'] == 'production'

module BitsService
  class App < Sinatra::Application
    use Routes::Buildpacks
  end
end
