require 'awspec'
require 'aws-sdk-core'
require 'aws-sdk-sts'
require 'git'

Awsecrets.load(secrets_path: File.expand_path('./secrets.yml', File.dirname(__FILE__)))

RSpec.configure do |config|
  config.expect_with :rspec do |expectations|
    expectations.include_chain_clauses_in_custom_matcher_descriptions = true
  end

  config.mock_with :rspec do |mocks|
    mocks.verify_partial_doubles = true
  end

  config.shared_context_metadata_behavior = :apply_to_host_groups

  if config.files_to_run.one?
    config.default_formatter = "doc"
  end
end

def buses(path)
  paths = Dir.glob("#{path}/**/*")
  paths.each do |file_path|
    if match = file_path.match(%r{bus/([^/]+)/([^/]+)})
      bus = match[1]
      rule = match[2].split(".")[0]
      yield(bus, rule) if block_given?
    end
  end
end

def sigv4_get_request(url)
  uri = URI(url)

  signer = Aws::Sigv4::Signer.new(
    service: 'execute-api',
    region: 'us-west-2',
    credentials_provider: Aws::SharedCredentials.new
  )

  request = Net::HTTP::Get.new(uri)
  signed_request = signer.sign_request(http_method: 'GET', url: uri.to_s)

  signed_request.headers.each { |key, value| request[key] = value }

  Net::HTTP.start(uri.host, uri.port, use_ssl: true) do |http|
    http.request(request)
  end
end

def bearer_get_request(uri)
  ssmc = Aws::SSM::Client.new
  creds = ssmc.get_parameter({
    name: "/linecard/auth0/client",
    with_decryption: true
  })


  auth0_uri = URI("https://linecard.us.auth0.com/oauth/token")
  http = Net::HTTP.new(url.host, url.port)
  http.use_ssl = true
  http.verify_mode = OpenSSL::SSL::VERIFY_NONE

  request = Net::HTTP::Post.new(url)
  request["content-type"] = 'application/json'
  request.body = creds.parameter.value

  response = http.request(request)
  token = JSON.parse(response.read_body)["access_token"]

  uri = URI(uri)
  request = Net::HTTP::Get.new(uri)
  request.headers["Authorization"] = "Bearer #{token}"
  Net::HTTP.start(uri.host, uri.port, use_ssl: true) do |http|
    http.request(request)
  end
end