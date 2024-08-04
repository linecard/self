require 'spec_helper'

raise "must provide FUNCTION_PATH in env" unless ENV.key?("FUNCTION_PATH")

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

  # Use default credential provider chain
  signer = Aws::Sigv4::Signer.new(
    service: 'execute-api',
    region: 'us-west-2',
    credentials_provider: Aws::SharedCredentials.new
  )

  # Create and sign the HTTP GET request
  request = Net::HTTP::Get.new(uri)
  signed_request = signer.sign_request(http_method: 'GET', url: uri.to_s)

  # Add signed headers to the request
  signed_request.headers.each { |key, value| request[key] = value }

  # Perform the HTTP request
  Net::HTTP.start(uri.host, uri.port, use_ssl: true) do |http|
    http.request(request)
  end
end

path = ENV["FUNCTION_PATH"]
name = File.basename(path)

describe name do
    # Git stuff
    owner = "linecard"
    repo = "self"
    git = Git.open(".")
    branch = git.current_branch
    sha = git.revparse("HEAD")
    origin = git.remote.url.gsub("git@github.com:", "https://github.com/")

    # Aws stuff
    region = "us-west-2"
    caller = Aws::STS::Client.new.get_caller_identity
    resource_name = "#{repo}-#{branch}-#{name}"
    function_name = resource_name
    function_arn = "arn:aws:lambda:#{region}:#{caller.account}:function:#{resource_name}"
    role_name = resource_name
    role_arn = "arn:aws:iam::#{caller.account}:role/#{resource_name}"
    policy_name = resource_name
    route_key = "ANY /#{repo}/#{branch}/#{name}/{proxy+}"
    url = "https://#{ENV["AWS_API_GATEWAY_ID"]}.execute-api.#{region}.amazonaws.com/#{repo}/#{branch}/#{name}/"

    describe lambda(function_name) do
      it { should exist }
      its(:function_name) { should eq function_name }
      its(:role) { should eq role_arn }
      it { should have_tag('Function').value(name) }
      it { should have_tag('Branch').value(branch) }
      it { should have_tag('Sha').value(sha) }
      it { should have_tag('Origin').value(origin) }
    end
    
    describe iam_role(role_name) do
        it { should exist }
        its(:assume_role_policy_document) { should include("AssumeRole") }
        its(:assume_role_policy_document) { should include("lambda.amazonaws.com") }
        it { should have_iam_policy(policy_name) }
        it { should have_tag('Function').value(name) }
        it { should have_tag('Branch').value(branch) }
        it { should have_tag('Sha').value(sha) }
        it { should have_tag('Origin').value(origin) }
    end

    describe iam_policy(policy_name) do
        it { should exist }
        it { should be_attached_to_role(role_arn) }
        its(:attachment_count) { should eq 1 }
        it { should have_tag('Function').value(name) }
        it { should have_tag('Branch').value(branch) }
        it { should have_tag('Sha').value(sha) }
        it { should have_tag('Origin').value(origin) }
    end

    if ENV.key?("AWS_API_GATEWAY_ID")
      describe apigatewayv2('self-verify') do
        it { should exist }
        it { should have_route_key(route_key).with_target(function_arn) }
      end

      describe "GET #{url}" do
        it "authenticated: return 200", retry: 10 do
          response = sigv4_get_request(url)
          expect(response.code).to eq("200")
        end

        it "unauthenticated: return 403", retry: 10 do
          response = Net::HTTP.get_response(URI(url))
          expect(response.code).to eq("403")
        end
      end
    end

    if !ENV.key?("AWS_API_GATEWAY_ID")
      describe apigatewayv2('self-verify') do
        it { should exist }
        it { should_not have_target(function_arn) }
      end
    end

    if ENV.key?("ENABLE_EVENTING_ON_DEPLOY")
      buses(path) do |bus, rule|
        describe eventbridge(bus) do
          it { should exist }
          it { should have_rule("#{resource_name}-#{rule}").with_target(function_arn) }
        end
      end
    end

    if !ENV.key?("ENABLE_EVENTING_ON_DEPLOY")
      buses(path) do |bus, rule|
        describe eventbridge(bus) do
          it { should exist }
          it { should_not have_rule("#{resource_name}-#{rule}") }
        end
      end
    end
end