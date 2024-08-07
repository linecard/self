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
