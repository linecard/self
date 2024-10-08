require 'spec_helper'

raise "must provide FUNCTION_PATH in env" unless ENV.key?("FUNCTION_PATH")

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
    url = "https://#{ENV["SELF_API_GATEWAY_ID"]}.execute-api.#{region}.amazonaws.com/#{repo}/#{branch}/#{name}/"

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

    if ENV.key?("SELF_API_GATEWAY_ID")
      describe apigatewayv2('self-verify') do
        it { should exist }
        it { should have_route_key(route_key).with_target(function_arn) }
      end

      if ENV.key?("SELF_API_GATEWAY_AUTH_TYPE") && ENV["SELF_API_GATEWAY_AUTH_TYPE"] == "AWS_IAM"
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

      if ENV.key?("SELF_API_GATEWAY_AUTH_TYPE") && ENV["SELF_API_GATEWAY_AUTH_TYPE"] == "JWT"
        describe "GET #{url}" do
          it "authenticated: return 200", retry: 10 do
            response = bearer_get_request(url)
            expect(response.code).to eq("200")
          end

          it "unauthenticated: return 401", retry: 10 do
            response = Net::HTTP.get_response(URI(url))
            expect(response.code).to eq("401")
          end
        end
      end
    end

    if !ENV.key?("SELF_API_GATEWAY_ID")
      describe apigatewayv2('self-verify') do
        it { should exist }
        it { should_not have_target(function_arn) }
      end
    end

    if ENV.key?("SELF_ENABLE_ON_DEPLOY")
      buses(path) do |bus, rule|
        describe eventbridge(bus) do
          it { should exist }
          it { should have_rule("#{resource_name}-#{rule}").with_target(function_arn) }
        end
      end
    end

    if ENV.key?("SELF_DISABLE_ON_DEPLOY")
      buses(path) do |bus, rule|
        describe eventbridge(bus) do
          it { should exist }
          it { should_not have_rule("#{resource_name}-#{rule}") }
        end
      end
    end

    if ENV.key?("SELF_SUBNET_IDS") && ENV.key?("SELF_SECURITY_GROUP_IDS")
      describe lambda(function_name) do
        its(:vpc_config) do
          expect(subject.vpc_config.subnet_ids).to eq ENV["SELF_SUBNET_IDS"].split(",")
        end

        its(:vpc_config) do
          expect(subject.vpc_config.security_group_ids).to eq ENV["SELF_SECURITY_GROUP_IDS"].split(",")
        end
      end
    else
      describe lambda(function_name) do
        its(:vpc_config) do
          expect(subject.vpc_config.subnet_ids).to be_empty
        end

        its(:vpc_config) do
          expect(subject.vpc_config.security_group_ids).to be_empty
        end
      end
    end
end