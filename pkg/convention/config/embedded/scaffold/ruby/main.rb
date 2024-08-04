require 'sinatra'
require 'json'

# Set content type globally
before do
  content_type :json
end

# Route for GET /
get '/' do
  headers.to_json
end

# Route for GET /healthz
get '/healthz' do
  { status: 'healthy' }.to_json
end

# Catch-all for 404 Not Found
not_found do
  { error: 'Not found' }.to_json
end

# Start the Sinatra server
set :bind, '0.0.0.0'
set :port, 8080