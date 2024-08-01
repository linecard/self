require 'spec_helper'

raise "must provide FUNCTION_PATH in env" unless ENV.key?("FUNCTION_PATH")

describe "Deployment" do
    path = ENV["FUNCTION_PATH"]
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
    route_key = "ANY /#{repo}/#{branch}/#{name}"

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
end


# Need to create eventbridge spec matchers in awspec fork

# def buses(path)
#   directory_pattern = "#{path}/*/*.*"
#   regex = %r{#{path}/([^/]+)/([^/]+)\.[^/]+}
#   file_paths = Dir.glob(directory_pattern)

#   file_paths.each do |file_path|
#     if match = file_path.match(regex)
#       bus = match[1]
#       rule = match[2]
#       yield(bus, rule) if block_given?
#     end
#   end
# end

# def bus_rule(bus, rule)
#   "Sid" => resource_name,
#   "Principal" => {"Service"=>"events.amazonaws.com"},
#   "Effect" => "Allow",
#   "Action" => "lambda:InvokeFunction",
#   "Resource" => function_arn,
#   "Condition" => {
#       "ArnLike" => {
#           "AWS:SourceArn" => "arn:aws:events:us-west-2:#{caller.account}:rule/#{resource_name}-#{bus}-#{rule}"
#       } 
#     }
# end