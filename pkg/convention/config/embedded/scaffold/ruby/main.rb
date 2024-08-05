require 'sinatra'
require 'json'

before do
  content_type :json
end

get '/' do
  headers.to_json
end

not_found do
  { error: 'Not found' }.to_json
end

set :bind, '0.0.0.0'
set :port, ENV["AWS_LWA_PORT"] || 8081