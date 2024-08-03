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

def functions(path)
  paths = Dir.glob("#{path}/**/*")
  paths.each do |file_path|
    if match = file_path.match(%r{init/([^/]+)})
      name = match[1]
      yield(name) if block_given?
    end
  end
end

functions.each do |name|
  describe "#{name}" do
      path = name
      name = File.basename(path)

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
          its(:assume_role_policy_document) do 
              should include("AssumeRole")
              should include("lambda.amazonaws.com")
          end
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
end