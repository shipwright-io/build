'''
before_step(context, step), after_step(context, step)
    These run before and after every step.
    The step passed in is an instance of Step.
before_feature(context, feature), after_feature(context, feature)
    These run before and after each feature file is exercised.
    The feature passed in is an instance of Feature.
before_all(context), after_all(context)
    These run before and after the whole shooting match.
'''

import subprocess
from pyshould import should

'''
before_scenario(context, scenario), after_scenario(context, scenario)
    These run before and after each scenario is run.
    The scenario passed in is an instance of Scenario.
'''

def before_scenario(_context, _scenario):
    print("\nGetting OC status before {} scenario".format(_scenario))
    code, output = subprocess.getstatusoutput('oc get project default')
    print("[CODE] {}".format(code))
    print("[CMD] {}".format(output))
    code | should.be_equal_to(0)
    print("***Connected to cluster***")
